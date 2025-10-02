export function connectWebSocket(token) {
	const ws = new WebSocket(`ws://${window.location.host}/ws`);

	ws.onopen = () => {
		ws.send(JSON.stringify({ type: "auth", token }));
	};

	return ws;
}
