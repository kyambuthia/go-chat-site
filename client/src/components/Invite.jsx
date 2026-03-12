import { useState } from "react";
import { sendInvite } from "../api";

export default function Invite({ compact = false, onInviteSent = null }) {
  const [username, setUsername] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState(null);
  const inputID = compact ? "invite-username-compact" : "invite-username";

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setMessage(null);
    const normalizedUsername = username.trim();
    try {
      await sendInvite(normalizedUsername);
      setMessage(`Invite sent to ${normalizedUsername}`);
      setUsername("");
      if (typeof onInviteSent === "function") {
        onInviteSent(normalizedUsername);
      }
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <div className={`invite ${compact ? "invite-compact" : ""}`}>
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor={inputID}>{compact ? "Invite by username" : "Username"}</label>
          <input
            id={inputID}
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Enter username"
            required
          />
        </div>
        <button type="submit" disabled={!username.trim()}>{compact ? "Invite" : "Send Invite"}</button>
      </form>
      {message && <p className="success-message">{message}</p>}
      {error && <p className="error-message">{error}</p>}
    </div>
  );
}
