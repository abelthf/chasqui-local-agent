# Ejemplos de integracion local

Esta carpeta contiene ejemplos de aplicaciones locales que reciben eventos desde `chasqui-local-agent` y envian respuestas usando la API local del agente.

## Ejemplo rapido con Python

Archivo directo: `python_app.py`

```bash
python3 python_app.py
```

## Ejemplo rapido con Node.js

Archivo directo: `node_app.js`

```bash
node node_app.js
```

## Estructura completa

- `python/app.py`: ejemplo Python con README propio.
- `node/app.js`: ejemplo Node.js con README propio.

## Como se conectan

1. La aplicacion local escucha en `http://localhost:5051/inbound`.
2. `chasqui-local-agent` recibe eventos desde Chasqui Cloud.
3. El agente llama a la aplicacion local.
4. La aplicacion local responde usando `POST http://localhost:5050/send`.

Ejecuta el agente apuntando al callback local:

```bash
CHASQUI_AGENT_TOKEN="agtkn_..." CHASQUI_API_KEY="sk_..." CHASQUI_AGENT_TRANSPORT="websocket" CHASQUI_LOCAL_CALLBACK_URL="http://localhost:5051/inbound" ../dist/chasqui-local-agent-1.0.0-linux-amd64
```
