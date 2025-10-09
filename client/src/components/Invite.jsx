import { useState } from "react";
import { sendInvite } from "../api";

export default function Invite() {
  const [username, setUsername] = useState("");
  const [message, setMessage] = useState("");

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      await sendInvite(username);
      setMessage(`Invite sent to ${username}`);
      setUsername("");
    } catch (error) {
      setMessage(error.message);
    }
  };

  return (
    <div className="invite">
      <h2>Invite a Friend</h2>
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="Enter username"
        />
        <button type="submit">Send Invite</button>
      </form>
      {message && <p>{message}</p>}
    </div>
  );
}
