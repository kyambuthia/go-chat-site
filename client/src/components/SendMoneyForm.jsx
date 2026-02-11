import { useMemo, useState } from "react";
import { sendMoney } from "../api";

const QUICK_AMOUNTS = [5, 10, 20, 50];

export default function SendMoneyForm({ onSendSuccess, onCancel, defaultRecipient }) {
  const [recipientUsername, setRecipientUsername] = useState(defaultRecipient || "");
  const [amount, setAmount] = useState("");
  const [status, setStatus] = useState({ type: "", message: "" });
  const [loading, setLoading] = useState(false);

  const parsedAmount = useMemo(() => parseFloat(amount), [amount]);

  const setError = (message) => setStatus({ type: "error", message });
  const setSuccess = (message) => setStatus({ type: "success", message });

  const handleSubmit = async (e) => {
    e.preventDefault();
    setStatus({ type: "", message: "" });

    if (!recipientUsername.trim()) {
      setError("Recipient username is required.");
      return;
    }

    if (Number.isNaN(parsedAmount) || parsedAmount <= 0) {
      setError("Enter a valid positive amount.");
      return;
    }

    setLoading(true);
    try {
      await sendMoney(recipientUsername.trim(), parsedAmount);
      setSuccess(`Sent $${parsedAmount.toFixed(2)} to ${recipientUsername.trim()}.`);
      setAmount("");
      if (!defaultRecipient) {
        setRecipientUsername("");
      }
      if (onSendSuccess) {
        onSendSuccess();
      }
    } catch (error) {
      setError(error.message || "Failed to send money.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="send-money-form">
      <div className="send-money-header">
        <h3>Send Money</h3>
        <p>Quick transfer to your contact.</p>
      </div>

      <form onSubmit={handleSubmit}>
        <div className="form-group">
          <label htmlFor="recipientUsername">Recipient</label>
          <input
            id="recipientUsername"
            type="text"
            value={recipientUsername}
            onChange={(e) => setRecipientUsername(e.target.value)}
            placeholder="username"
            required
            disabled={!!defaultRecipient || loading}
          />
        </div>

        <div className="form-group">
          <label htmlFor="amount">Amount (USD)</label>
          <input
            id="amount"
            type="number"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="0.00"
            required
            min="0.01"
            step="0.01"
            inputMode="decimal"
            disabled={loading}
          />
          <div className="quick-amounts" role="group" aria-label="Quick amount selection">
            {QUICK_AMOUNTS.map((quick) => (
              <button
                key={quick}
                type="button"
                className="quick-amount-btn"
                onClick={() => setAmount(String(quick))}
                disabled={loading}
              >
                ${quick}
              </button>
            ))}
          </div>
        </div>

        <div className="send-money-actions">
          <button type="submit" disabled={loading}>{loading ? "Sending..." : "Send"}</button>
          <button type="button" onClick={onCancel} disabled={loading} className="danger">Cancel</button>
        </div>
      </form>

      {status.message && (
        <p className={`money-status ${status.type === "error" ? "is-error" : "is-success"}`}>{status.message}</p>
      )}
    </div>
  );
}
