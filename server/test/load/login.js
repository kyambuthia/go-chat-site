import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: 10,
  duration: "30s",
};

export default function () {
  const payload = JSON.stringify({ username: "load-user", password: "password123" });
  const res = http.post("http://localhost:8080/api/login", payload, {
    headers: { "Content-Type": "application/json" },
  });

  check(res, {
    "status is 200 or 401": (r) => r.status === 200 || r.status === 401,
  });

  sleep(1);
}
