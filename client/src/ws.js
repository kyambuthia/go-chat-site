export function connectWebSocket(token) {
	const ws = new WebSocket(`ws://${window.location.host}/ws?token=${token}`);

	ws.onopen = () => {
		console.log("WebSocket connected");
	};

	return ws;
}
