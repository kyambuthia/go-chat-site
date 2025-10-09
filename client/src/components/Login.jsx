import { useState } from "react";
import { loginUser } from "../api";

export default function Login({ onLogin, onShowRegister }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const data = await loginUser(username, password);
      if (data && data.token) {
        onLogin(data.token);
      } else {
        setMessage("Login failed: No token received.");
      }
    } catch (error) {
      setMessage(error.message);
    }
  };

  return (
    <div className="auth-container">
      <h2>Login</h2>
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="Username"
          required
        />
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Password"
          required
        />
        <button type="submit" className="primary">Login</button>
      </form>
      {message && <p>{message}</p>}
      <button onClick={onShowRegister} className="auth-switch-button">
        Don't have an account? Register
      </button>
    </div>
  );
}