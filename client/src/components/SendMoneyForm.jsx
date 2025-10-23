import { useState } from "react";
import { sendMoney } from "../api";

export default function SendMoneyForm({ onSendSuccess, onCancel, defaultRecipient }) {
  const [recipientUsername, setRecipientUsername] = useState(defaultRecipient || "");
  const [amount, setAmount] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setMessage("");
    setLoading(true);

    try {
      const parsedAmount = parseFloat(amount);
      if (isNaN(parsedAmount) || parsedAmount <= 0) {
        setMessage("Please enter a valid positive amount.");
        setLoading(false);
        return;
      }

      await sendMoney(recipientUsername, parsedAmount);
      setMessage("Money sent successfully!");
      setRecipientUsername("");
      setAmount("");
      if (onSendSuccess) {
        onSendSuccess();
      }
    } catch (error) {
      setMessage(`Error: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="send-money-form">
      <h3>Send Money</h3>
      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="recipientUsername">Recipient Username</label>
          <input
            id="recipientUsername"
            type="text"
            value={recipientUsername}
            onChange={(e) => setRecipientUsername(e.target.value)}
            placeholder="Enter recipient's username"
            required
            disabled={!!defaultRecipient}
          />
        </div>
        <div className="form-group">
          <label htmlFor="amount">Amount</label>
          <input
            id="amount"
            type="number"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="Enter amount"
            required
            min="0.01"
            step="0.01"
          />
        </div>
        <button type="submit" disabled={loading}>
          {loading ? "Sending..." : "Send"}
        </button>
        <button type="button" onClick={onCancel} disabled={loading} className="danger">
          Cancel
        </button>
      </form>
      {message && <p className="message">{message}</p>}
    </div>
  );
}
