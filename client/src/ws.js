export function connectWebSocket(token) {
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const wsUrl = `${protocol}://${window.location.host}/ws`;
  const subprotocols = token ? [`bearer.${token}`] : undefined;

  const ws = new WebSocket(wsUrl, subprotocols);

  ws.onopen = () => {
    console.log("WebSocket connected");
  };

  return ws;
}
