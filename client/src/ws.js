import { buildWebSocketURL } from "./lib/runtimeConfig";

export function connectWebSocket(token) {
  const wsUrl = buildWebSocketURL({
    wsBaseURL: import.meta.env.VITE_WS_URL,
    apiBaseURL: import.meta.env.VITE_API_BASE_URL,
  });
  const subprotocols = token ? [`bearer.${token}`] : undefined;

  const ws = new WebSocket(wsUrl, subprotocols);

  ws.onopen = () => {
    console.log("WebSocket connected");
  };

  return ws;
}
