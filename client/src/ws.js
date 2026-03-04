import { buildWebSocketURL, getViteEnv } from "./lib/runtimeConfig.js";

export function connectWebSocket(token) {
  const wsUrl = buildWebSocketURL({
    wsBaseURL: getViteEnv("VITE_WS_URL"),
    apiBaseURL: getViteEnv("VITE_API_BASE_URL"),
  });
  const subprotocols = token ? [`bearer.${token}`] : undefined;

  const ws = new WebSocket(wsUrl, subprotocols);

  return ws;
}
