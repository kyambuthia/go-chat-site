import { useState } from "react";
import { registerUser } from "../api";

export default function Register({ onRegisterSuccess }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      await registerUser(username, password);
      setMessage("Registration successful! Please log in.");
      setUsername("");
      setPassword("");
      onRegisterSuccess();
    } catch (error) {
      setMessage(error.message);
    }
  };

  return (
    <div className="auth-container">
      <h2>Register</h2>
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
        <button type="submit" className="primary">Register</button>
      </form>
      {message && <p>{message}</p>}
      <button onClick={onRegisterSuccess} className="auth-switch-button">
        Already have an account? Login
      </button>
    </div>
  );
}