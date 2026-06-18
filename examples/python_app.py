#!/usr/bin/env python3
"""
Aplicacion local de ejemplo para Chasqui Local Agent.

Recibe eventos en http://localhost:5051/inbound y responde usando
http://localhost:5050/send del agente local.

Ejecutar:
    python3 app.py
"""

import json
import urllib.request
from http.server import BaseHTTPRequestHandler, HTTPServer

AGENT_SEND_URL = "http://localhost:5050/send"
LISTEN_ADDR = ("127.0.0.1", 5051)


def send_message(to: str, message: str, idempotency_key: str | None = None) -> None:
    payload = {"to": to, "message": message}
    if idempotency_key:
        payload["idempotency_key"] = idempotency_key

    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        AGENT_SEND_URL,
        data=data,
        method="POST",
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=15) as resp:
        resp.read()


def handle_inbound(event: dict) -> None:
    payload = event.get("payload", {})
    sender = payload.get("sender_phone")
    text = (payload.get("text") or "").strip().lower()
    event_id = event.get("id")

    print(f"evento={event_id} sender={sender} text={text!r}")

    if not sender:
        return

    if text in ("hola", "hello", "buenas"):
        reply = "Hola, soy tu aplicacion local conectada por Chasqui Local Agent."
    elif text == "estado":
        reply = "La aplicacion local esta activa."
    else:
        reply = "Mensaje recibido por la aplicacion local."

    send_message(sender, reply, idempotency_key=f"local-reply-{event_id}")


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b'{"status":"ok"}')
            return
        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        if self.path != "/inbound":
            self.send_response(404)
            self.end_headers()
            return

        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length)
        try:
            event = json.loads(body.decode("utf-8"))
            handle_inbound(event)
        except Exception as exc:
            print(f"error procesando evento: {exc}")
            self.send_response(500)
            self.end_headers()
            return

        self.send_response(204)
        self.end_headers()

    def log_message(self, fmt, *args):
        return


if __name__ == "__main__":
    server = HTTPServer(LISTEN_ADDR, Handler)
    print(f"Aplicacion local escuchando en http://{LISTEN_ADDR[0]}:{LISTEN_ADDR[1]}/inbound")
    server.serve_forever()
