import { it, expect, mock } from "bun:test";
import { connectWebSocket } from "../src/ws";

global.WebSocket = mock(function (url) {
	return {
		url,
		onopen: () => {},
		send: () => {},
	};
});

it("connectWebSocket returns a WebSocket instance", () => {
	const ws = connectWebSocket("test-token");
	expect(ws.url).toBe(`ws://${window.location.host}/ws`);
});
