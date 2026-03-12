import { useEffect, useState } from "react";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";
import { PersonIcon, Pencil2Icon } from "@radix-ui/react-icons";
import { getMe, getWallet, getWalletTransfers, updateMe } from "../api";
import SendMoneyForm from "./SendMoneyForm";

export default function AccountPage({ handleLogout }) {
  const [user, setUser] = useState(null);
  const [wallet, setWallet] = useState(null);
  const [transfers, setTransfers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [profileForm, setProfileForm] = useState({ display_name: "", avatar_url: "" });
  const [profileStatus, setProfileStatus] = useState({ type: "", message: "" });
  const [savingProfile, setSavingProfile] = useState(false);
  const [showSendMoney, setShowSendMoney] = useState(false);

  useEffect(() => {
    let cancelled = false;

    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        const [userResponse, walletResponse, transfersResponse] = await Promise.all([
          getMe(),
          getWallet(),
          getWalletTransfers({ limit: 10 }),
        ]);
        if (cancelled) {
          return;
        }
        setUser(userResponse);
        setWallet(walletResponse);
        setTransfers(transfersResponse || []);
        setProfileForm({
          display_name: userResponse?.display_name || "",
          avatar_url: userResponse?.avatar_url || "",
        });
      } catch (err) {
        if (!cancelled) {
          setError(err.message);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    fetchData();

    return () => {
      cancelled = true;
    };
  }, []);

  const refreshWalletData = async () => {
    const [walletResponse, transfersResponse] = await Promise.all([
      getWallet(),
      getWalletTransfers({ limit: 10 }),
    ]);
    setWallet(walletResponse);
    setTransfers(transfersResponse || []);
  };

  const handleProfileChange = (event) => {
    const { name, value } = event.target;
    setProfileForm((current) => ({ ...current, [name]: value }));
  };

  const handleProfileSubmit = async (event) => {
    event.preventDefault();
    setSavingProfile(true);
    setProfileStatus({ type: "", message: "" });

    try {
      const updatedUser = await updateMe({
        display_name: profileForm.display_name.trim(),
        avatar_url: profileForm.avatar_url.trim(),
      });
      setUser(updatedUser);
      setProfileForm({
        display_name: updatedUser.display_name || "",
        avatar_url: updatedUser.avatar_url || "",
      });
      setProfileStatus({ type: "success", message: "Profile updated." });
    } catch (err) {
      setProfileStatus({ type: "error", message: err.message || "Failed to update profile." });
    } finally {
      setSavingProfile(false);
    }
  };

  const handleTransferSuccess = async () => {
    await refreshWalletData();
  };

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
          <Avatar className="avatar-placeholder account-avatar">
            <AvatarImage src={user.avatar_url || ""} alt={user.display_name || user.username} />
            <AvatarFallback><PersonIcon width="24" height="24" /></AvatarFallback>
          </Avatar>
          <div className="user-info">
            <h3>{user.display_name || user.username}</h3>
            <p>@{user.username}</p>
          </div>
        </div>
      )}
      <form className="profile-card" onSubmit={handleProfileSubmit}>
        <div className="card-heading">
          <div>
            <h4>Edit Profile</h4>
            <p>Update the name and avatar shown around the app.</p>
          </div>
          <Pencil2Icon />
        </div>
        <div className="form-group">
          <label htmlFor="display_name">Display Name</label>
          <input
            id="display_name"
            name="display_name"
            type="text"
            value={profileForm.display_name}
            onChange={handleProfileChange}
            placeholder="How contacts should see you"
            disabled={savingProfile}
          />
        </div>
        <div className="form-group">
          <label htmlFor="avatar_url">Avatar URL</label>
          <input
            id="avatar_url"
            name="avatar_url"
            type="url"
            value={profileForm.avatar_url}
            onChange={handleProfileChange}
            placeholder="https://example.com/avatar.png"
            disabled={savingProfile}
          />
        </div>
        <div className="profile-actions">
          <button type="submit" disabled={savingProfile}>
            {savingProfile ? "Saving..." : "Save Profile"}
          </button>
        </div>
        {profileStatus.message && (
          <p className={`money-status ${profileStatus.type === "error" ? "is-error" : "is-success"}`}>
            {profileStatus.message}
          </p>
        )}
      </form>
      {wallet && (
        <div className="wallet-card">
          <div className="card-heading">
            <div>
              <h4>Wallet Balance</h4>
              <p className="wallet-balance">${wallet.balance ? wallet.balance.toFixed(2) : "0.00"}</p>
            </div>
            <button type="button" className="secondary-button" onClick={() => setShowSendMoney((current) => !current)}>
              {showSendMoney ? "Hide Transfer" : "Send Money"}
            </button>
          </div>
        </div>
      )}
      {showSendMoney && (
        <SendMoneyForm
          onSendSuccess={handleTransferSuccess}
          onCancel={() => setShowSendMoney(false)}
        />
      )}
      <div className="wallet-card transfer-history-card">
        <div className="card-heading">
          <div>
            <h4>Recent Transfers</h4>
            <p>Sent and received wallet activity.</p>
          </div>
        </div>
        {transfers.length === 0 ? (
          <p className="empty-chat-message">No transfers yet.</p>
        ) : (
          <ul className="transfer-history">
            {transfers.map((transfer) => {
              const counterpartyName = transfer.counterparty_display_name || transfer.counterparty_username;
              const amountLabel = `${transfer.direction === "received" ? "+" : "-"}$${Number(transfer.amount || 0).toFixed(2)}`;

              return (
                <li key={transfer.id} className="transfer-history-item">
                  <Avatar className="avatar-placeholder transfer-avatar">
                    <AvatarImage src={transfer.counterparty_avatar_url || ""} alt={counterpartyName} />
                    <AvatarFallback><PersonIcon width="20" height="20" /></AvatarFallback>
                  </Avatar>
                  <div className="transfer-copy">
                    <strong>{transfer.direction === "received" ? "Received from" : "Sent to"} {counterpartyName}</strong>
                    <span>@{transfer.counterparty_username}</span>
                    <time dateTime={transfer.created_at}>{new Date(transfer.created_at).toLocaleString()}</time>
                  </div>
                  <span className={`transfer-amount ${transfer.direction === "received" ? "is-positive" : "is-negative"}`}>
                    {amountLabel}
                  </span>
                </li>
              );
            })}
          </ul>
        )}
      </div>
      <button onClick={handleLogout} className="logout-button danger">
        Logout
      </button>
    </div>
  );
}
