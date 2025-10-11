import { useState } from "react";
import { loginUser } from "../api";
import { Label } from '@radix-ui/react-label';

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
        <div className="form-group">
          <Label htmlFor="username">Username</Label>
          <input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Enter your username"
            required
          />
        </div>
        <div className="form-group">
          <Label htmlFor="password">Password</Label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Enter your password"
            required
          />
        </div>
        <button type="submit">Login</button>
      </form>
      {message && <p className="error-message">{message}</p>}
      <button onClick={onShowRegister} className="auth-switch-button">
        Don't have an account? Register
      </button>
    </div>
  );
}