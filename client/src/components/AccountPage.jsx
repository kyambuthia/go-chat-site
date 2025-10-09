import { useState, useEffect } from "react";
import { getMe, getWallet } from "../api";

export default function AccountPage({ handleLogout }) {
  const [user, setUser] = useState(null);
  const [wallet, setWallet] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        const userResponse = await getMe();
        const walletResponse = await getWallet();
        setUser(userResponse);
        setWallet(walletResponse);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  if (loading) {
    return <div className="account-page">Loading account details...</div>;
  }

  if (error) {
    return <div className="account-page">Error: {error}</div>;
  }

  return (
    <div className="account-page">
      <h2>My Account</h2>
      {user && (
        <div className="user-details">
          <p>
            <strong>Username:</strong> {user.username}
          </p>
          {user.display_name && (
            <p>
              <strong>Display Name:</strong> {user.display_name}
            </p>
          )}
          {user.avatar_url && (
            <p>
              <strong>Avatar:</strong> <img src={user.avatar_url} alt="Avatar" className="avatar-preview" />
            </p>
          )}
        </div>
      )}
      {wallet && (
        <div className="wallet-details">
          <p>
            <strong>Balance:</strong> ${wallet.balance ? wallet.balance.toFixed(2) : "0.00"}
          </p>
        </div>
      )}
      <button onClick={handleLogout} className="logout-button danger">
        Logout
      </button>
    </div>
  );
}