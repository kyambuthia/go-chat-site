import ws from "k6/ws";
import { check } from "k6";

export const options = {
  vus: 5,
  duration: "20s",
};

export default function () {
  const token = __ENV.JWT_TOKEN || "invalid";
  const url = `ws://localhost:8080/ws`;
  const params = {
    headers: {
      Authorization: `Bearer ${token}`,
      Origin: "http://localhost:5173",
    },
  };

  const res = ws.connect(url, params, (socket) => {
    socket.on("open", () => {
      socket.close();
    });
  });

  check(res, {
    "handshake status is expected": (r) => r && (r.status === 101 || r.status === 401 || r.status === 429),
  });
}
