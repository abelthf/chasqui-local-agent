# Ejemplo Python

Este ejemplo representa la aplicacion local del cliente.

## Ejecutar

Terminal 1: aplicacion local

```bash
python3 app.py
```

Terminal 2: Chasqui Local Agent

```bash
CHASQUI_AGENT_TOKEN="agtkn_..." \
CHASQUI_API_KEY="sk_..." \
CHASQUI_AGENT_TRANSPORT="websocket" \
CHASQUI_LOCAL_CALLBACK_URL="http://localhost:5051/inbound" \
../../dist/chasqui-local-agent-1.0.0-linux-amd64
```

Cuando llegue un mensaje entrante, el agente llamara a `/inbound` y esta app respondera usando `http://localhost:5050/send`.
