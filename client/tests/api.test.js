import { it, expect, mock } from "bun:test";
import { registerUser, loginUser } from "../src/api";

global.fetch = mock(async (url, options) => {
	if (url === "/api/register") {
		const body = await new Response(options.body).json();
		if (body.username === "testuser") {
			return new Response(JSON.stringify({ id: 1, username: "testuser" }), {
				status: 201,
			});
		}
	} else if (url === "/api/login") {
		const body = await new Response(options.body).json();
		if (body.username === "testuser" && body.password === "password123") {
			return new Response(JSON.stringify({ token: "test-token" }), {
				status: 200,
			});
		}
	}
	return new Response(null, { status: 404 });
});

it("registerUser calls fetch with the correct payload", async () => {
	const response = await registerUser("testuser", "password123");
	expect(response.username).toBe("testuser");
});

it("loginUser calls fetch with the correct payload", async () => {
	const response = await loginUser("testuser", "password123");
	expect(response.token).toBe("test-token");
});
