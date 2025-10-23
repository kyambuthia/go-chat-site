import { useState, useEffect } from "react";
import { getMe, getWallet } from "../api";
import { PersonIcon } from '@radix-ui/react-icons';

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
        <div className="user-card">
          <div className="avatar-placeholder">
            <PersonIcon width="24" height="24" />
          </div>
          <div className="user-info">
            <h3>{user.display_name || user.username}</h3>
            <p>@{user.username}</p>
          </div>
        </div>
      )}
      {wallet && (
        <div className="wallet-card">
          <h4>Wallet Balance</h4>
          <p>${wallet.balance ? wallet.balance.toFixed(2) : "0.00"}</p>
        </div>
      )}
      <button onClick={handleLogout} className="logout-button danger">
        Logout
      </button>
    </div>
  );
}