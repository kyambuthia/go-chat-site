import { useState, useEffect } from "react";
import Chat from "./components/Chat";
import Login from "./components/Login";
import Register from "./components/Register";
import ContactsPage from "./components/ContactsPage";
import InvitesPage from "./components/InvitesPage";
import AccountPage from "./components/AccountPage"; // Import AccountPage
import { setToken } from "./api";

const WS_URL = "ws://localhost:5173/ws";

function App() {
  const [ws, setWs] = useState(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [showRegister, setShowRegister] = useState(false);
  const [selectedContact, setSelectedContact] = useState(null);
  const [activeTab, setActiveTab] = useState("chat");

  useEffect(() => {
    if (isLoggedIn) {
      const token = localStorage.getItem("token");
      const socket = new WebSocket(`${WS_URL}?token=${token}`);
      setWs(socket);

      return () => {
        socket.close();
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
        return <Chat ws={ws} selectedContact={selectedContact} setSelectedContact={setSelectedContact} />;
      case "contacts":
        return <ContactsPage setSelectedContact={setSelectedContact} />;
      case "invites":
        return <InvitesPage />;
      case "account": // Add account case
        return <AccountPage handleLogout={handleLogout} />;
      default:
        return <Chat ws={ws} selectedContact={selectedContact} setSelectedContact={setSelectedContact} />;
    }
  };

  return (
    <div className="App">
      <main className="app-content">
        {renderContent()}
      </main>
      {isLoggedIn && (
        <nav>
          <button onClick={() => setActiveTab("chat")} className={activeTab === "chat" ? "active" : ""}>
            Chat
          </button>
          <button onClick={() => setActiveTab("contacts")} className={activeTab === "contacts" ? "active" : ""}>
            Contacts
          </button>
          <button onClick={() => setActiveTab("invites")} className={activeTab === "invites" ? "active" : ""}>
            Invites
          </button>
          <button onClick={() => setActiveTab("account")} className={activeTab === "account" ? "active" : ""}>
            Account
          </button>
        </nav>
      )}
    </div>
  );
}

export default App;
