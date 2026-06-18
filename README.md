# Chasqui Local Agent

Agente local para conectar aplicaciones privadas con Chasqui Cloud sin IP publica, sin abrir puertos y sin tuneles. Toda la comunicacion se inicia desde la maquina local hacia Chasqui Cloud por HTTPS/WebSocket.

## Modalidades de uso

Puedes usar el agente de cuatro formas:

1. Binario precompilado para Linux, Windows o macOS.
2. Docker Compose.
3. Docker manual.
4. Build local desde codigo fuente Go.

Para recepcion inmediata usa `CHASQUI_AGENT_TRANSPORT=websocket`. Para fallback simple usa `CHASQUI_AGENT_TRANSPORT=poll`.

## Requisitos

- Un `agent_token` emitido desde Chasqui Cloud.
- Una aplicacion local que reciba eventos entrantes, por ejemplo `http://localhost:5051/inbound`.
- Opcional: una API key de Chasqui si la aplicacion local enviara mensajes salientes mediante el agente.

## Variables

- `CHASQUI_AGENT_TOKEN`: token Bearer emitido por Chasqui Cloud.
- `CHASQUI_BASE_URL`: URL base, por defecto `https://chasqui.inkalab.org.pe/api`.
- `CHASQUI_AGENT_TRANSPORT`: `websocket` para recepcion inmediata o `poll` como fallback.
- `CHASQUI_LOCAL_CALLBACK_URL`: endpoint local de tu aplicacion para eventos entrantes.
- `CHASQUI_API_KEY`: API key para que `/send` pueda enviar mensajes.
- `CHASQUI_AGENT_ID`: identificador de instancia.
- `CHASQUI_AGENT_SECRET`: secreto opcional enviado al callback local.
- `CHASQUI_AGENT_DB`: ruta SQLite local.
- `CHASQUI_AGENT_LISTEN_ADDR`: direccion de escucha local, por defecto `127.0.0.1:5050`.

## Opcion 1: binario precompilado

Los binarios se publican en `dist/`:

- `chasqui-local-agent-1.0.0-linux-amd64`
- `chasqui-local-agent-1.0.0-linux-arm64`
- `chasqui-local-agent-1.0.0-windows-amd64.exe`
- `chasqui-local-agent-1.0.0-darwin-amd64`
- `chasqui-local-agent-1.0.0-darwin-arm64`
- `SHA256SUMS.txt`

### Linux

```bash
chmod +x chasqui-local-agent-1.0.0-linux-amd64
export CHASQUI_AGENT_TOKEN="agtkn_..."
export CHASQUI_AGENT_TRANSPORT="websocket"
export CHASQUI_LOCAL_CALLBACK_URL="http://localhost:5051/inbound"
export CHASQUI_API_KEY="sk_..."
./chasqui-local-agent-1.0.0-linux-amd64
```

### macOS Intel

```bash
chmod +x chasqui-local-agent-1.0.0-darwin-amd64
export CHASQUI_AGENT_TOKEN="agtkn_..."
export CHASQUI_AGENT_TRANSPORT="websocket"
export CHASQUI_LOCAL_CALLBACK_URL="http://localhost:5051/inbound"
./chasqui-local-agent-1.0.0-darwin-amd64
```

### macOS Apple Silicon

```bash
chmod +x chasqui-local-agent-1.0.0-darwin-arm64
export CHASQUI_AGENT_TOKEN="agtkn_..."
export CHASQUI_AGENT_TRANSPORT="websocket"
export CHASQUI_LOCAL_CALLBACK_URL="http://localhost:5051/inbound"
./chasqui-local-agent-1.0.0-darwin-arm64
```

### Windows PowerShell

```powershell
$env:CHASQUI_AGENT_TOKEN="agtkn_..."
$env:CHASQUI_AGENT_TRANSPORT="websocket"
$env:CHASQUI_LOCAL_CALLBACK_URL="http://localhost:5051/inbound"
$env:CHASQUI_API_KEY="sk_..."
.\chasqui-local-agent-1.0.0-windows-amd64.exe
```

## Opcion 2: Docker Compose

Crea un archivo `.env` en esta carpeta:

```env
CHASQUI_BASE_URL=https://chasqui.inkalab.org.pe/api
CHASQUI_AGENT_TOKEN=agtkn_...
CHASQUI_API_KEY=sk_...
CHASQUI_LOCAL_CALLBACK_URL=http://host.docker.internal:5051/inbound
CHASQUI_AGENT_ID=local-agent
CHASQUI_AGENT_SECRET=
CHASQUI_AGENT_TRANSPORT=websocket
CHASQUI_AGENT_LISTEN_ADDR=0.0.0.0:5050
```

Levanta el agente:

```bash
docker compose up -d --build
```

Ver logs:

```bash
docker compose logs -f
```

Detener:

```bash
docker compose down
```

Nota para Linux: si `host.docker.internal` no resuelve, configura la URL callback con la IP accesible del host o agrega `extra_hosts` en `docker-compose.yml`.

## Opcion 3: Docker manual

Build:

```bash
docker build -t chasqui-local-agent:1.0.0 .
```

Run:

```bash
docker run -d --name chasqui-local-agent \
  -p 127.0.0.1:5050:5050 \
  -v chasqui_agent_data:/data \
  -e CHASQUI_BASE_URL="https://chasqui.inkalab.org.pe/api" \
  -e CHASQUI_AGENT_TOKEN="agtkn_..." \
  -e CHASQUI_API_KEY="sk_..." \
  -e CHASQUI_AGENT_TRANSPORT="websocket" \
  -e CHASQUI_LOCAL_CALLBACK_URL="http://host.docker.internal:5051/inbound" \
  -e CHASQUI_AGENT_DB="/data/agent-events.db" \
  -e CHASQUI_AGENT_LISTEN_ADDR="0.0.0.0:5050" \
  chasqui-local-agent:1.0.0
```

## Opcion 4: build local desde fuente

Requiere Go 1.25 o superior.

```bash
go build -o chasqui-local-agent .
export CHASQUI_AGENT_TOKEN="agtkn_..."
export CHASQUI_AGENT_TRANSPORT="websocket"
export CHASQUI_LOCAL_CALLBACK_URL="http://localhost:5051/inbound"
./chasqui-local-agent
```


## Ejemplos de aplicacion local

Incluye ejemplos listos para entender como recibir eventos y enviar respuestas:

- [examples/python](examples/python): servidor local Python sin dependencias externas.
- [examples/node](examples/node): servidor local Node.js sin paquetes externos.

Flujo recomendado para probar:

Terminal 1, levanta tu aplicacion local:

```bash
cd examples/python
python3 app.py
```

Terminal 2, levanta el agente apuntando al callback local:

```bash
CHASQUI_AGENT_TOKEN="agtkn_..." \
CHASQUI_API_KEY="sk_..." \
CHASQUI_AGENT_TRANSPORT="websocket" \
CHASQUI_LOCAL_CALLBACK_URL="http://localhost:5051/inbound" \
./chasqui-local-agent
```

Cuando Chasqui Cloud reciba un mensaje entrante, el agente hara `POST` a `http://localhost:5051/inbound`. La aplicacion de ejemplo procesara el evento y respondera llamando a `http://localhost:5050/send`.

## API local

El agente escucha por defecto solo en loopback: `http://127.0.0.1:5050`. En Docker se publica tambien solo en loopback del host.

- `GET /health`: salud basica.
- `GET /status`: estado local, version y eventos pendientes locales.
- `POST /send`: envia un mensaje saliente a Chasqui Cloud.
- `POST /inbound`: alias de `/send` por compatibilidad.

Ejemplo de envio saliente:

```bash
curl -X POST http://localhost:5050/send \
  -H "Content-Type: application/json" \
  -d '{
    "to": "51999999999",
    "message": "Hola desde mi aplicacion local",
    "idempotency_key": "pedido-123"
  }'
```

## Callback hacia la aplicacion local

Cuando llega un evento desde Chasqui Cloud, el agente hace `POST` a `CHASQUI_LOCAL_CALLBACK_URL` con un payload similar:

```json
{
  "id": 123,
  "event_type": "message.inbound",
  "payload": {
    "message_id": 456,
    "conversation_id": "51999999999",
    "sender_phone": "51999999999",
    "text": "Hola",
    "is_group": false
  },
  "idempotency_key": "pmid:abc",
  "created_at": "2026-06-18T20:00:00Z"
}
```

Tu aplicacion local debe responder `2xx`. Si responde error o no esta disponible, el evento queda pendiente para reintento.

## Build de binarios release

```bash
VERSION=1.0.0 ./scripts/build-release.sh
```

Genera artefactos en `dist/` para Linux, Windows y macOS, mas `SHA256SUMS.txt`.

## Verificacion de checksums

Linux/macOS:

```bash
cd dist
sha256sum -c SHA256SUMS.txt
```

Windows PowerShell:

```powershell
Get-FileHash .\chasqui-local-agent-1.0.0-windows-amd64.exe -Algorithm SHA256
```

Compara el hash con `SHA256SUMS.txt`.
