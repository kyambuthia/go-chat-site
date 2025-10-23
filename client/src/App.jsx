import { useState, useEffect } from "react";
import Chat from "./components/Chat";
import Login from "./components/Login";
import Register from "./components/Register";
import ContactsPage from "./components/ContactsPage";
import InvitesPage from "./components/InvitesPage";
import AccountPage from "./components/AccountPage";
import { setToken, getInvites } from "./api";
import { ChatBubbleIcon, PersonIcon, EnvelopeClosedIcon, GearIcon } from '@radix-ui/react-icons';

import { connectWebSocket } from "./ws";

function App() {
  const [ws, setWs] = useState(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [showRegister, setShowRegister] = useState(false);
  const [selectedContact, setSelectedContact] = useState(null);
  const [activeTab, setActiveTab] = useState("chat");
  const [onlineUsers, setOnlineUsers] = useState([]);
  const [invites, setInvites] = useState([]);

  const fetchInvites = async () => {
    try {
      const response = await getInvites();
      setInvites(response || []);
    } catch (error) {
      console.error("Failed to fetch invites:", error);
    }
  };

  useEffect(() => {
    if (isLoggedIn) {
      const token = localStorage.getItem("token");
      const socket = connectWebSocket(token);
      setWs(socket);

      socket.onmessage = (event) => {
        const message = JSON.parse(event.data);
        if (message.type === "user_online") {
          setOnlineUsers((prevOnlineUsers) => [...prevOnlineUsers, message.from]);
        } else if (message.type === "user_offline") {
          setOnlineUsers((prevOnlineUsers) =>
            prevOnlineUsers.filter((user) => user !== message.from)
          );
        }
      };

      fetchInvites();
      const intervalId = setInterval(fetchInvites, 5000);

      return () => {
        socket.close();
        clearInterval(intervalId);
      };
    }
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
      return <div>Connecting...</div>;
    }

    switch (activeTab) {
      case "chat":
        return <Chat ws={ws} selectedContact={selectedContact} setSelectedContact={setSelectedContact} onlineUsers={onlineUsers} />;
      case "contacts":
        return <ContactsPage setSelectedContact={setSelectedContact} onlineUsers={onlineUsers} />;
      case "invites":
        return <InvitesPage invites={invites} onUpdate={fetchInvites} />;
      case "account":
        return <AccountPage handleLogout={handleLogout} />;
      default:
        return <Chat ws={ws} selectedContact={selectedContact} setSelectedContact={setSelectedContact} onlineUsers={onlineUsers} />;
    }
  };

  return (
    <div className="App">
      <main className={`app-content ${selectedContact ? 'no-padding' : ''}`}>
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
