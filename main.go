package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
)

var version = "dev"

type Config struct {
	BaseURL      string
	AgentToken   string
	APIKey       string
	CallbackURL  string
	AgentID      string
	AgentSecret  string
	ListenAddr   string
	DBPath       string
	UseWebSocket bool
}

type AgentEvent struct {
	ID             int64                  `json:"id"`
	EventType      string                 `json:"event_type"`
	Payload        map[string]interface{} `json:"payload"`
	IdempotencyKey string                 `json:"idempotency_key"`
	CreatedAt      string                 `json:"created_at"`
	ExpiresAt      *string                `json:"expires_at"`
}

type EventBatch struct {
	Events []AgentEvent `json:"events"`
	Count  int          `json:"count"`
}

type SendRequest struct {
	To             string `json:"to"`
	Message        string `json:"message"`
	MediaURL       string `json:"media_url,omitempty"`
	MediaType      string `json:"media_type,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type Agent struct {
	cfg        Config
	db         *sql.DB
	httpClient *http.Client
	logger     *slog.Logger
	startedAt  time.Time
	lastSeen   atomic.Value
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := loadConfig()
	db, err := openDB(cfg.DBPath)
	if err != nil {
		logger.Error("local_store_unavailable")
		os.Exit(1)
	}
	agent := &Agent{cfg: cfg, db: db, httpClient: &http.Client{Timeout: 20 * time.Second}, logger: logger, startedAt: time.Now()}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := agent.register(ctx); err != nil {
		logger.Error("agent_register_failed")
		os.Exit(1)
	}

	go agent.serveLocal(ctx)
	go agent.heartbeatLoop(ctx)
	if cfg.UseWebSocket {
		go agent.websocketLoop(ctx)
	} else {
		go agent.pollLoop(ctx)
	}

	logger.Info("local_agent_started", "agent_id", cfg.AgentID, "listen_addr", cfg.ListenAddr, "version", version)
	<-ctx.Done()
	logger.Info("local_agent_stopped")
}

func loadConfig() Config {
	host, _ := os.Hostname()
	return Config{
		BaseURL:      strings.TrimRight(env("CHASQUI_BASE_URL", "https://chasqui.inkalab.org.pe/api"), "/"),
		AgentToken:   os.Getenv("CHASQUI_AGENT_TOKEN"),
		APIKey:       os.Getenv("CHASQUI_API_KEY"),
		CallbackURL:  env("CHASQUI_LOCAL_CALLBACK_URL", "http://localhost:5051/inbound"),
		AgentID:      env("CHASQUI_AGENT_ID", host),
		AgentSecret:  os.Getenv("CHASQUI_AGENT_SECRET"),
		ListenAddr:   env("CHASQUI_AGENT_LISTEN_ADDR", "127.0.0.1:5050"),
		DBPath:       env("CHASQUI_AGENT_DB", "agent-events.db"),
		UseWebSocket: env("CHASQUI_AGENT_TRANSPORT", "poll") == "websocket",
	}
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`create table if not exists events (
		id integer primary key,
		event_type text not null,
		payload_json text not null,
		status text not null,
		created_at text,
		updated_at text not null
	);`)
	return db, err
}

func (a *Agent) register(ctx context.Context) error {
	if a.cfg.AgentToken == "" {
		return errors.New("agent_token_required")
	}
	body := map[string]string{"agent_id": a.cfg.AgentID}
	return a.doJSON(ctx, http.MethodPost, "/agent/register", body, true, nil)
}

func (a *Agent) heartbeatLoop(ctx context.Context) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = a.heartbeat(ctx)
		}
	}
}

func (a *Agent) heartbeat(ctx context.Context) error {
	var out map[string]interface{}
	err := a.doJSON(ctx, http.MethodPost, "/agent/heartbeat", nil, true, &out)
	if err == nil {
		a.lastSeen.Store(time.Now().Format(time.RFC3339))
	}
	return err
}

func (a *Agent) pollLoop(ctx context.Context) {
	for attempt := 0; ; attempt++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := a.pullAndProcess(ctx); err != nil {
			a.logger.Warn("agent_poll_failed")
			time.Sleep(backoff(attempt))
			continue
		}
		attempt = 0
		time.Sleep(5 * time.Second)
	}
}

func (a *Agent) websocketLoop(ctx context.Context) {
	for attempt := 0; ; attempt++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := a.runWebSocket(ctx); err != nil {
			a.logger.Warn("agent_websocket_reconnect")
			time.Sleep(backoff(attempt))
			continue
		}
		attempt = 0
	}
}

func (a *Agent) runWebSocket(ctx context.Context) error {
	u := strings.Replace(a.cfg.BaseURL, "https://", "wss://", 1)
	u = strings.Replace(u, "http://", "ws://", 1)
	u = strings.TrimSuffix(u, "/api") + "/ws/agent"
	header := http.Header{"Authorization": {"Bearer " + a.cfg.AgentToken}, "Sec-WebSocket-Extensions": {"permessage-deflate"}}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u, header)
	if err != nil {
		return err
	}
	defer conn.Close()
	for {
		var msg struct {
			Type   string       `json:"type"`
			Events []AgentEvent `json:"events"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			return err
		}
		if msg.Type == "events" {
			_ = a.processEventsWithAck(ctx, msg.Events, func(eventID int64) error {
				return conn.WriteJSON(map[string]interface{}{"type": "ack", "event_id": eventID})
			})
		}
	}
}

func (a *Agent) pullAndProcess(ctx context.Context) error {
	var batch EventBatch
	if err := a.doJSON(ctx, http.MethodGet, "/agent/events/pending", nil, true, &batch); err != nil {
		return err
	}
	return a.processEvents(ctx, batch.Events)
}

func (a *Agent) processEvents(ctx context.Context, events []AgentEvent) error {
	return a.processEventsWithAck(ctx, events, func(eventID int64) error {
		return a.ack(ctx, eventID)
	})
}

func (a *Agent) processEventsWithAck(ctx context.Context, events []AgentEvent, ackFn func(int64) error) error {
	for _, ev := range events {
		_ = a.storeEvent(ev, "delivered")
		if err := a.forwardToLocal(ctx, ev); err != nil {
			a.logger.Warn("local_callback_failed", "event_id", ev.ID)
			continue
		}
		if err := ackFn(ev.ID); err != nil {
			a.logger.Warn("agent_ack_failed", "event_id", ev.ID)
			continue
		}
		_ = a.storeEvent(ev, "acknowledged")
	}
	return nil
}

func (a *Agent) forwardToLocal(ctx context.Context, ev AgentEvent) error {
	b, _ := json.Marshal(ev)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.CallbackURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.cfg.AgentSecret != "" {
		req.Header.Set("X-Chasqui-Agent-Secret", a.cfg.AgentSecret)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("local_callback_rejected")
	}
	return nil
}

func (a *Agent) ack(ctx context.Context, id int64) error {
	return a.doJSON(ctx, http.MethodPost, fmt.Sprintf("/agent/events/%d/ack", id), nil, true, nil)
}

func (a *Agent) storeEvent(ev AgentEvent, status string) error {
	payload, _ := json.Marshal(ev.Payload)
	_, err := a.db.Exec(`insert into events(id,event_type,payload_json,status,created_at,updated_at)
	values(?,?,?,?,?,?) on conflict(id) do update set status=excluded.status, updated_at=excluded.updated_at`,
		ev.ID, ev.EventType, string(payload), status, ev.CreatedAt, time.Now().Format(time.RFC3339))
	return err
}

func (a *Agent) serveLocal(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		var pending int
		_ = a.db.QueryRow(`select count(*) from events where status != 'acknowledged'`).Scan(&pending)
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "agent_id": a.cfg.AgentID, "version": version, "started_at": a.startedAt.Format(time.RFC3339), "last_seen": a.lastSeen.Load(), "pending_local_events": pending})
	})
	sendHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body SendRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid_payload", 400)
			return
		}
		var out map[string]interface{}
		if err := a.doJSON(r.Context(), http.MethodPost, "/message/send", body, false, &out); err != nil {
			http.Error(w, "delivery_failed", 502)
			return
		}
		json.NewEncoder(w).Encode(out)
	}
	mux.HandleFunc("/inbound", sendHandler)
	mux.HandleFunc("/send", sendHandler)
	srv := &http.Server{Addr: a.cfg.ListenAddr, Handler: mux}
	go func() { <-ctx.Done(); _ = srv.Shutdown(context.Background()) }()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.logger.Error("local_api_stopped")
	}
}

func (a *Agent) doJSON(ctx context.Context, method, path string, body interface{}, agentAuth bool, out interface{}) error {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.cfg.BaseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if agentAuth {
		req.Header.Set("Authorization", "Bearer "+a.cfg.AgentToken)
	} else if a.cfg.APIKey != "" {
		req.Header.Set("X-API-Key", a.cfg.APIKey)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("cloud_request_failed")
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func backoff(attempt int) time.Duration {
	if attempt > 6 {
		attempt = 6
	}
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func wsURL(base string) string {
	u, _ := url.Parse(base)
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/ws/agent"
	return u.String()
}
