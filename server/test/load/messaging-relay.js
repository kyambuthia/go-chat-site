import ws from "k6/ws";
import { check, sleep } from "k6";

export const options = {
  vus: 2,
  iterations: 20,
};

export default function () {
  const senderToken = __ENV.SENDER_JWT || "invalid";
  const recipientUsername = __ENV.RECIPIENT_USERNAME || "bob";

  const res = ws.connect("ws://localhost:8080/ws", {
    headers: {
      Authorization: `Bearer ${senderToken}`,
      Origin: "http://localhost:5173",
    },
  }, (socket) => {
    socket.on("open", () => {
      socket.send(JSON.stringify({
        type: "direct_message",
        to: recipientUsername,
        body: `load message ${Date.now()}`,
      }));
      sleep(0.2);
      socket.close();
    });
  });

  check(res, {
    "relay connect status is expected": (r) => r && (r.status === 101 || r.status === 401 || r.status === 429),
  });
}
