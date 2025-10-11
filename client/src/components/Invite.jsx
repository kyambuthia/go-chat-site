import { useState } from "react";
import { sendInvite } from "../api";

export default function Invite() {
  const [username, setUsername] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState(null);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setMessage(null);
    try {
      await sendInvite(username);
      setMessage(`Invite sent to ${username}`);
      setUsername("");
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <div className="invite">
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="username">Username</label>
          <input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Enter username"
            required
          />
        </div>
        <button type="submit">Send Invite</button>
      </form>
      {message && <p className="success-message">{message}</p>}
      {error && <p className="error-message">{error}</p>}
    </div>
  );
}