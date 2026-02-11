import { useState, useEffect } from "react";
import Chat from "./components/Chat";
import Login from "./components/Login";
import Register from "./components/Register";
import ContactsPage from "./components/ContactsPage";
import InvitesPage from "./components/InvitesPage";
import AccountPage from "./components/AccountPage";
import { setToken, getInvites } from "./api";
import { ChatBubbleIcon, PersonIcon, EnvelopeClosedIcon, GearIcon } from "@radix-ui/react-icons";

import { connectWebSocket } from "./ws";

function App() {
  const [ws, setWs] = useState(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [showRegister, setShowRegister] = useState(false);
  const [selectedContact, setSelectedContact] = useState(null);
  const [activeTab, setActiveTab] = useState("chat");
  const [onlineUsers, setOnlineUsers] = useState([]);
  const [invites, setInvites] = useState([]);
  const [lastWsMessage, setLastWsMessage] = useState(null);
  const [wsStatus, setWsStatus] = useState("offline");

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (token) {
      setToken(token);
      setIsLoggedIn(true);
    }
  }, []);

  const fetchInvites = async () => {
    try {
      const response = await getInvites();
      setInvites(response || []);
    } catch (error) {
      console.error("Failed to fetch invites:", error);
    }
  };

  useEffect(() => {
    if (!isLoggedIn) {
      return;
    }

    setWsStatus("connecting");
    const token = localStorage.getItem("token");
    const socket = connectWebSocket(token);
    setWs(socket);

    socket.onopen = () => {
      setWsStatus("online");
    };

    socket.onclose = () => {
      setWsStatus("offline");
    };

    socket.onerror = () => {
      setWsStatus("offline");
    };

    socket.onmessage = (event) => {
      const message = JSON.parse(event.data);
      if (message.type === "user_online") {
        setOnlineUsers((prevOnlineUsers) =>
          prevOnlineUsers.includes(message.from) ? prevOnlineUsers : [...prevOnlineUsers, message.from]
        );
        return;
      }
      if (message.type === "user_offline") {
        setOnlineUsers((prevOnlineUsers) => prevOnlineUsers.filter((user) => user !== message.from));
        return;
      }
      setLastWsMessage(message);
    };

    fetchInvites();
    const intervalId = setInterval(fetchInvites, 7000);

    return () => {
      socket.close();
      clearInterval(intervalId);
    };
  }, [isLoggedIn]);

  const handleLogin = (token) => {
    localStorage.setItem("token", token);
    setToken(token);
    setIsLoggedIn(true);
    setActiveTab("chat");
  };

  const handleLogout = () => {
    localStorage.removeItem("token");
    setToken(null);
    setIsLoggedIn(false);
    setWs(null);
    setOnlineUsers([]);
    setLastWsMessage(null);
    setSelectedContact(null);
    setInvites([]);
    setWsStatus("offline");
  };

  const renderContent = () => {
    if (!isLoggedIn) {
      return showRegister ? (
        <Register onRegisterSuccess={() => setShowRegister(false)} />
      ) : (
        <Login onLogin={handleLogin} onShowRegister={() => setShowRegister(true)} />
      );
    }

    if (!ws) {
      return <div className="status-block">Connecting...</div>;
    }

    switch (activeTab) {
      case "chat":
        return (
          <Chat
            ws={ws}
            selectedContact={selectedContact}
            setSelectedContact={setSelectedContact}
            onlineUsers={onlineUsers}
            lastWsMessage={lastWsMessage}
          />
        );
      case "contacts":
        return <ContactsPage setSelectedContact={setSelectedContact} onlineUsers={onlineUsers} />;
      case "invites":
        return <InvitesPage invites={invites} onUpdate={fetchInvites} />;
      case "account":
        return <AccountPage handleLogout={handleLogout} />;
      default:
        return (
          <Chat
            ws={ws}
            selectedContact={selectedContact}
            setSelectedContact={setSelectedContact}
            onlineUsers={onlineUsers}
            lastWsMessage={lastWsMessage}
          />
        );
    }
  };

  const isContactsView = !selectedContact && (activeTab === "chat" || activeTab === "contacts");

  return (
    <div className="App">
      <main className={`app-content ${selectedContact ? "no-padding" : ""} ${isContactsView ? "contacts-view" : ""}`}>
        {isLoggedIn && !selectedContact && (
          <div className={`connection-pill ${wsStatus}`}>
            <span className="connection-dot" />
            {wsStatus === "online" ? "Connected" : wsStatus === "connecting" ? "Connecting" : "Offline"}
          </div>
        )}
        {renderContent()}
      </main>
      {isLoggedIn && !selectedContact && (
        <nav>
          <button onClick={() => setActiveTab("chat")} className={`nav-item ${activeTab === "chat" ? "active" : ""}`}>
            <div className="nav-icon-wrapper"><ChatBubbleIcon /></div>
            <span>Chat</span>
          </button>
          <button onClick={() => setActiveTab("contacts")} className={`nav-item ${activeTab === "contacts" ? "active" : ""}`}>
            <div className="nav-icon-wrapper"><PersonIcon /></div>
            <span>Contacts</span>
          </button>
          <button onClick={() => setActiveTab("invites")} className={`nav-item ${activeTab === "invites" ? "active" : ""}`}>
            <div className="nav-icon-wrapper">
              <EnvelopeClosedIcon />
              {invites.length > 0 && <div className="notification-dot"></div>}
            </div>
            <span>Invites</span>
          </button>
          <button onClick={() => setActiveTab("account")} className={`nav-item ${activeTab === "account" ? "active" : ""}`}>
            <div className="nav-icon-wrapper"><GearIcon /></div>
            <span>Account</span>
          </button>
        </nav>
      )}
    </div>
  );
}

export default App;
