#!/usr/bin/env node
/*
Aplicacion local de ejemplo para Chasqui Local Agent.

Recibe eventos en http://localhost:5051/inbound y responde usando
http://localhost:5050/send del agente local.

Ejecutar:
    node app.js
*/

const http = require("http");

const AGENT_SEND_URL = "http://localhost:5050/send";
const PORT = 5051;

function readBody(req) {
  return new Promise((resolve, reject) => {
    let body = "";
    req.on("data", (chunk) => { body += chunk; });
    req.on("end", () => resolve(body));
    req.on("error", reject);
  });
}

async function sendMessage(to, message, idempotencyKey) {
  const payload = JSON.stringify({
    to,
    message,
    idempotency_key: idempotencyKey,
  });

  const res = await fetch(AGENT_SEND_URL, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: payload,
  });

  if (!res.ok) {
    throw new Error(`agent_send_failed:${res.status}`);
  }
}

async function handleInbound(event) {
  const payload = event.payload || {};
  const sender = payload.sender_phone;
  const text = String(payload.text || "").trim().toLowerCase();
  const eventId = event.id;

  console.log({ event_id: eventId, sender, text });

  if (!sender) return;

  let reply = "Mensaje recibido por la aplicacion local.";
  if (["hola", "hello", "buenas"].includes(text)) {
    reply = "Hola, soy tu aplicacion local conectada por Chasqui Local Agent.";
  } else if (text === "estado") {
    reply = "La aplicacion local esta activa.";
  }

  await sendMessage(sender, reply, `local-reply-${eventId}`);
}

const server = http.createServer(async (req, res) => {
  if (req.method === "GET" && req.url === "/health") {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ status: "ok" }));
    return;
  }

  if (req.method !== "POST" || req.url !== "/inbound") {
    res.writeHead(404);
    res.end();
    return;
  }

  try {
    const raw = await readBody(req);
    await handleInbound(JSON.parse(raw));
    res.writeHead(204);
    res.end();
  } catch (err) {
    console.error(err);
    res.writeHead(500);
    res.end("local_processing_failed");
  }
});

server.listen(PORT, "127.0.0.1", () => {
  console.log(`Aplicacion local escuchando en http://127.0.0.1:${PORT}/inbound`);
});
