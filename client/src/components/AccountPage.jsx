import { useEffect, useState } from "react";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";
import { PersonIcon, Pencil2Icon } from "@radix-ui/react-icons";
import {
  getDevices,
  getMe,
  getSessions,
  getWallet,
  getWalletTransfers,
  publishDevicePrekeys,
  registerDeviceIdentity,
  revokeDeviceIdentity,
  revokeSession,
  updateMe,
} from "../api";
import {
  DEFAULT_DEVICE_ALGORITHM,
  formatPrekeysForTextarea,
  generateDeviceIdentityBundle,
} from "../lib/deviceIdentity.js";
import {
  listLocalDeviceBundleIDs,
  removeLocalDeviceBundle,
  saveLocalDeviceBundle,
} from "../lib/deviceKeyStore.js";
import SendMoneyForm from "./SendMoneyForm";

const emptyDeviceForm = () => ({
  label: "",
  algorithm: DEFAULT_DEVICE_ALGORITHM,
  identity_key: "",
  signed_prekey_id: "",
  signed_prekey: "",
  signed_prekey_signature: "",
  prekeys_text: "",
});

const parsePrekeysInput = (value) => {
  const lines = value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);

  return lines.map((line) => {
    const separatorIndex = line.includes(":") ? line.indexOf(":") : line.indexOf(",");
    if (separatorIndex <= 0 || separatorIndex >= line.length - 1) {
      throw new Error("Prekeys must be entered one per line as id:key.");
    }

    const prekeyID = Number(line.slice(0, separatorIndex).trim());
    if (!Number.isInteger(prekeyID) || prekeyID <= 0) {
      throw new Error("Each prekey line must start with a positive integer id.");
    }

    const publicKey = line.slice(separatorIndex + 1).trim();
    if (!publicKey) {
      throw new Error("Each prekey line must include a public key.");
    }

    return { prekey_id: prekeyID, public_key: publicKey };
  });
};

const formatKeyPreview = (value) => {
  if (!value) {
    return "Not published";
  }
  if (value.length <= 28) {
    return value;
  }
  return `${value.slice(0, 14)}...${value.slice(-10)}`;
};

export default function AccountPage({ handleLogout }) {
  const [user, setUser] = useState(null);
  const [wallet, setWallet] = useState(null);
  const [transfers, setTransfers] = useState([]);
  const [sessions, setSessions] = useState([]);
  const [devices, setDevices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [profileForm, setProfileForm] = useState({ display_name: "", avatar_url: "" });
  const [profileStatus, setProfileStatus] = useState({ type: "", message: "" });
  const [sessionStatus, setSessionStatus] = useState({ type: "", message: "" });
  const [deviceForm, setDeviceForm] = useState(emptyDeviceForm);
  const [deviceStatus, setDeviceStatus] = useState({ type: "", message: "" });
  const [pendingLocalBundle, setPendingLocalBundle] = useState(null);
  const [localBundleDeviceIDs, setLocalBundleDeviceIDs] = useState([]);
  const [publishInputs, setPublishInputs] = useState({});
  const [savingProfile, setSavingProfile] = useState(false);
  const [savingDevice, setSavingDevice] = useState(false);
  const [generatingBundle, setGeneratingBundle] = useState(false);
  const [revokingSessionID, setRevokingSessionID] = useState(null);
  const [revokingDeviceID, setRevokingDeviceID] = useState(null);
  const [publishingDeviceID, setPublishingDeviceID] = useState(null);
  const [showSendMoney, setShowSendMoney] = useState(false);

  useEffect(() => {
    let cancelled = false;

    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        const [userResponse, walletResponse, transfersResponse, sessionsResponse, devicesResponse] = await Promise.all([
          getMe(),
          getWallet(),
          getWalletTransfers({ limit: 10 }),
          getSessions(),
          getDevices(),
        ]);
        if (cancelled) {
          return;
        }
        setUser(userResponse);
        setWallet(walletResponse);
        setTransfers(transfersResponse || []);
        setSessions(sessionsResponse || []);
        setDevices(devicesResponse || []);
        setLocalBundleDeviceIDs(listLocalDeviceBundleIDs());
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

  const refreshSessions = async () => {
    const sessionResponse = await getSessions();
    setSessions(sessionResponse || []);
  };

  const refreshDevices = async () => {
    const deviceResponse = await getDevices();
    setDevices(deviceResponse || []);
    setLocalBundleDeviceIDs(listLocalDeviceBundleIDs());
  };

  const handleProfileChange = (event) => {
    const { name, value } = event.target;
    setProfileForm((current) => ({ ...current, [name]: value }));
  };

  const handleDeviceFormChange = (event) => {
    const { name, value } = event.target;
    if (name !== "label" && pendingLocalBundle) {
      setPendingLocalBundle(null);
    }
    setDeviceForm((current) => ({ ...current, [name]: value }));
  };

  const handlePublishInputChange = (deviceID, value) => {
    setPublishInputs((current) => ({ ...current, [deviceID]: value }));
  };

  const handleGenerateDeviceBundle = async () => {
    setGeneratingBundle(true);
    setDeviceStatus({ type: "", message: "" });

    try {
      const bundle = await generateDeviceIdentityBundle({
        algorithm: deviceForm.algorithm.trim() || DEFAULT_DEVICE_ALGORITHM,
      });
      setPendingLocalBundle(bundle.private_bundle);
      setDeviceForm((current) => ({
        ...current,
        algorithm: bundle.algorithm,
        identity_key: bundle.identity_key,
        signed_prekey_id: String(bundle.signed_prekey_id),
        signed_prekey: bundle.signed_prekey,
        signed_prekey_signature: bundle.signed_prekey_signature,
        prekeys_text: formatPrekeysForTextarea(bundle.prekeys),
      }));
      setDeviceStatus({ type: "success", message: "Generated a local device bundle. Register the device to save its private keys in this browser." });
    } catch (err) {
      setDeviceStatus({ type: "error", message: err.message || "Failed to generate a local device bundle." });
    } finally {
      setGeneratingBundle(false);
    }
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

  const handleRevokeSession = async (sessionID) => {
    setRevokingSessionID(sessionID);
    setSessionStatus({ type: "", message: "" });

    try {
      await revokeSession(sessionID);
      await refreshSessions();
      setSessionStatus({ type: "success", message: "Session revoked." });
    } catch (err) {
      setSessionStatus({ type: "error", message: err.message || "Failed to revoke session." });
    } finally {
      setRevokingSessionID(null);
    }
  };

  const handleRegisterDevice = async (event) => {
    event.preventDefault();
    setSavingDevice(true);
    setDeviceStatus({ type: "", message: "" });

    try {
      const prekeys = parsePrekeysInput(deviceForm.prekeys_text);
      const device = await registerDeviceIdentity({
        label: deviceForm.label.trim(),
        algorithm: deviceForm.algorithm.trim() || DEFAULT_DEVICE_ALGORITHM,
        identity_key: deviceForm.identity_key.trim(),
        signed_prekey_id: Number(deviceForm.signed_prekey_id),
        signed_prekey: deviceForm.signed_prekey.trim(),
        signed_prekey_signature: deviceForm.signed_prekey_signature.trim(),
        prekeys,
      });
      if (pendingLocalBundle && device?.id) {
        saveLocalDeviceBundle({
          userID: user?.id || 0,
          deviceID: device.id,
          bundle: pendingLocalBundle,
        });
      }
      await refreshDevices();
      setDeviceForm(emptyDeviceForm());
      setPendingLocalBundle(null);
      setDeviceStatus({
        type: "success",
        message: pendingLocalBundle && device?.id
          ? "Device identity registered and private keys saved locally."
          : "Device identity registered.",
      });
    } catch (err) {
      setDeviceStatus({ type: "error", message: err.message || "Failed to register device identity." });
    } finally {
      setSavingDevice(false);
    }
  };

  const handlePublishPrekeys = async (event, deviceID) => {
    event.preventDefault();
    setPublishingDeviceID(deviceID);
    setDeviceStatus({ type: "", message: "" });

    try {
      const prekeys = parsePrekeysInput(publishInputs[deviceID] || "");
      await publishDevicePrekeys(deviceID, prekeys);
      await refreshDevices();
      setPublishInputs((current) => ({ ...current, [deviceID]: "" }));
      setDeviceStatus({ type: "success", message: "Prekeys published." });
    } catch (err) {
      setDeviceStatus({ type: "error", message: err.message || "Failed to publish prekeys." });
    } finally {
      setPublishingDeviceID(null);
    }
  };

  const handleRevokeDevice = async (deviceID) => {
    setRevokingDeviceID(deviceID);
    setDeviceStatus({ type: "", message: "" });

    try {
      await revokeDeviceIdentity(deviceID);
      removeLocalDeviceBundle(deviceID);
      await refreshDevices();
      setDeviceStatus({ type: "success", message: "Device identity revoked." });
    } catch (err) {
      setDeviceStatus({ type: "error", message: err.message || "Failed to revoke device identity." });
    } finally {
      setRevokingDeviceID(null);
    }
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
      <div className="wallet-card session-card">
        <div className="card-heading">
          <div>
            <h4>Device Sessions</h4>
            <p>Review active devices and revoke access per session.</p>
          </div>
        </div>
        {sessions.length === 0 ? (
          <p className="empty-chat-message">No active sessions found.</p>
        ) : (
          <ul className="session-list">
            {sessions.map((session) => (
              <li key={session.id} className="session-item">
                <div className="session-copy">
                  <div className="session-title-row">
                    <strong>{session.device_label || "This device"}</strong>
                    {session.current && <span className="session-badge">Current</span>}
                  </div>
                  <span>{session.user_agent || "Unknown client"}</span>
                  <span>Last IP: {session.last_seen_ip || "unknown"}</span>
                  <time dateTime={session.last_seen_at || session.created_at}>
                    Last seen {new Date(session.last_seen_at || session.created_at).toLocaleString()}
                  </time>
                </div>
                {!session.current && (
                  <button
                    type="button"
                    className="secondary-button"
                    onClick={() => handleRevokeSession(session.id)}
                    disabled={revokingSessionID === session.id}
                  >
                    {revokingSessionID === session.id ? "Revoking..." : "Revoke"}
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
        {sessionStatus.message && (
          <p className={`money-status ${sessionStatus.type === "error" ? "is-error" : "is-success"}`}>
            {sessionStatus.message}
          </p>
        )}
      </div>
      <div className="wallet-card device-card">
        <div className="card-heading">
          <div>
            <h4>Device Identities</h4>
            <p>Register public device keys now so encrypted messaging can use them later.</p>
          </div>
        </div>
        <form className="device-form" onSubmit={handleRegisterDevice}>
          <div className="profile-actions">
            <button
              type="button"
              className="secondary-button"
              onClick={handleGenerateDeviceBundle}
              disabled={savingDevice || generatingBundle}
            >
              {generatingBundle ? "Generating..." : "Generate Local Bundle"}
            </button>
          </div>
          <div className="form-group">
            <label htmlFor="device_label">Label</label>
            <input
              id="device_label"
              name="label"
              type="text"
              value={deviceForm.label}
              onChange={handleDeviceFormChange}
              placeholder="Phone, laptop, tablet"
              disabled={savingDevice}
            />
          </div>
          <div className="device-form-grid">
            <div className="form-group">
              <label htmlFor="device_algorithm">Algorithm</label>
              <input
                id="device_algorithm"
                name="algorithm"
                type="text"
                value={deviceForm.algorithm}
                onChange={handleDeviceFormChange}
                disabled={savingDevice}
              />
            </div>
            <div className="form-group">
              <label htmlFor="signed_prekey_id">Signed Prekey ID</label>
              <input
                id="signed_prekey_id"
                name="signed_prekey_id"
                type="number"
                min="1"
                value={deviceForm.signed_prekey_id}
                onChange={handleDeviceFormChange}
                placeholder="1"
                disabled={savingDevice}
              />
            </div>
          </div>
          <div className="form-group">
            <label htmlFor="identity_key">Identity Key</label>
            <textarea
              id="identity_key"
              name="identity_key"
              rows="3"
              value={deviceForm.identity_key}
              onChange={handleDeviceFormChange}
              placeholder="Public identity key"
              disabled={savingDevice}
            />
          </div>
          <div className="form-group">
            <label htmlFor="signed_prekey">Signed Prekey</label>
            <textarea
              id="signed_prekey"
              name="signed_prekey"
              rows="3"
              value={deviceForm.signed_prekey}
              onChange={handleDeviceFormChange}
              placeholder="Current signed prekey public value"
              disabled={savingDevice}
            />
          </div>
          <div className="form-group">
            <label htmlFor="signed_prekey_signature">Signed Prekey Signature</label>
            <textarea
              id="signed_prekey_signature"
              name="signed_prekey_signature"
              rows="2"
              value={deviceForm.signed_prekey_signature}
              onChange={handleDeviceFormChange}
              placeholder="Signature over the signed prekey"
              disabled={savingDevice}
            />
          </div>
          <div className="form-group">
            <label htmlFor="device_prekeys_text">Prekeys</label>
            <textarea
              id="device_prekeys_text"
              name="prekeys_text"
              rows="4"
              value={deviceForm.prekeys_text}
              onChange={handleDeviceFormChange}
              placeholder={"One per line as id:key\n1:base64-public-key\n2:base64-public-key"}
              disabled={savingDevice}
            />
            <p className="device-hint">Enter public prekeys one per line using <code>id:key</code>.</p>
          </div>
          <div className="profile-actions">
            <button type="submit" disabled={savingDevice}>
              {savingDevice ? "Registering..." : "Register Device Identity"}
            </button>
          </div>
        </form>
        {devices.length === 0 ? (
          <p className="empty-chat-message">No device identities registered yet.</p>
        ) : (
          <ul className="device-list">
            {devices.map((device) => (
              <li key={device.id} className="device-item">
                <div className="device-copy">
                  <div className="session-title-row">
                    <strong>{device.label || "This device"}</strong>
                    {device.current_session && <span className="session-badge">Current Session</span>}
                    {localBundleDeviceIDs.includes(device.id) && <span className="session-badge">Local Keys</span>}
                    <span className={`session-badge ${device.state === "revoked" ? "device-badge-revoked" : ""}`}>
                      {device.state}
                    </span>
                  </div>
                  <span>Algorithm: {device.algorithm}</span>
                  <span>Active prekeys: {device.prekey_count}</span>
                  <span className="device-key-preview">Identity key: {formatKeyPreview(device.identity_key)}</span>
                  <span className="device-key-preview">Signed prekey: #{device.signed_prekey_id} {formatKeyPreview(device.signed_prekey)}</span>
                  <time dateTime={device.rotated_at || device.created_at}>
                    Last rotated {new Date(device.rotated_at || device.created_at).toLocaleString()}
                  </time>
                  {device.revoked_at && (
                    <time dateTime={device.revoked_at}>
                      Revoked {new Date(device.revoked_at).toLocaleString()}
                    </time>
                  )}
                </div>
                <div className="device-actions">
                  {device.state !== "revoked" && (
                    <form className="device-prekeys-form" onSubmit={(event) => handlePublishPrekeys(event, device.id)}>
                      <label htmlFor={`publish-prekeys-${device.id}`}>Publish More Prekeys</label>
                      <textarea
                        id={`publish-prekeys-${device.id}`}
                        rows="3"
                        value={publishInputs[device.id] || ""}
                        onChange={(event) => handlePublishInputChange(device.id, event.target.value)}
                        placeholder={"One per line as id:key\n101:base64-public-key"}
                        disabled={publishingDeviceID === device.id}
                      />
                      <button type="submit" className="secondary-button" disabled={publishingDeviceID === device.id}>
                        {publishingDeviceID === device.id ? "Publishing..." : "Publish Prekeys"}
                      </button>
                    </form>
                  )}
                  {device.state !== "revoked" && (
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={() => handleRevokeDevice(device.id)}
                      disabled={revokingDeviceID === device.id}
                    >
                      {revokingDeviceID === device.id ? "Revoking..." : "Revoke Device"}
                    </button>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
        {deviceStatus.message && (
          <p className={`money-status ${deviceStatus.type === "error" ? "is-error" : "is-success"}`}>
            {deviceStatus.message}
          </p>
        )}
      </div>
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
