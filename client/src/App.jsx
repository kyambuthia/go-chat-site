import { useState, useEffect, useRef } from "react";
import Chat from "./components/Chat";
import Login from "./components/Login";
import Register from "./components/Register";
import ContactsPage from "./components/ContactsPage";
import AccountPage from "./components/AccountPage";
import { setToken, getInvites, setAuthErrorHandler } from "./api";
import { ChatBubbleIcon, PersonIcon, GearIcon } from "@radix-ui/react-icons";

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

  const reconnectAttemptRef = useRef(0);
  const reconnectTimerRef = useRef(null);
  const activeSocketRef = useRef(null);

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

    let cancelled = false;

    const clearReconnect = () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    };

    const scheduleReconnect = () => {
      clearReconnect();
      const attempt = reconnectAttemptRef.current;
      const delay = Math.min(1000 * (2 ** attempt), 15000);
      reconnectAttemptRef.current = attempt + 1;

      reconnectTimerRef.current = setTimeout(() => {
        if (!cancelled) {
          connect();
        }
      }, delay);
    };

    const connect = () => {
      if (cancelled) {
        return;
      }

      setWsStatus("connecting");
      const token = localStorage.getItem("token");
      const socket = connectWebSocket(token);
      activeSocketRef.current = socket;
      setWs(socket);

      socket.onopen = () => {
        reconnectAttemptRef.current = 0;
        setWsStatus("online");
      };

      socket.onclose = () => {
        if (cancelled) {
          return;
        }
        setWsStatus("offline");
        scheduleReconnect();
      };

      socket.onerror = () => {
        setWsStatus("offline");
      };

      socket.onmessage = (event) => {
        let message;
        try {
          message = JSON.parse(event.data);
        } catch (err) {
          console.error("Received invalid WebSocket payload:", err);
          return;
        }
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
    };

    connect();
    fetchInvites();
    const intervalId = setInterval(fetchInvites, 7000);

    return () => {
      cancelled = true;
      clearReconnect();
      clearInterval(intervalId);
      if (activeSocketRef.current) {
        activeSocketRef.current.close();
        activeSocketRef.current = null;
      }
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
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    reconnectAttemptRef.current = 0;
    if (activeSocketRef.current) {
      activeSocketRef.current.close();
      activeSocketRef.current = null;
    }
    setWs(null);
    setOnlineUsers([]);
    setLastWsMessage(null);
    setSelectedContact(null);
    setInvites([]);
    setWsStatus("offline");
  };

  useEffect(() => {
    setAuthErrorHandler((message) => {
      if (message === "invalid token") {
        handleLogout();
      }
    });

    return () => {
      setAuthErrorHandler(null);
    };
  }, []);

  const renderContent = () => {
    if (!isLoggedIn) {
      return showRegister ? (
        <Register onRegisterSuccess={() => setShowRegister(false)} />
      ) : (
        <Login onLogin={handleLogin} onShowRegister={() => setShowRegister(true)} />
      );
    }

    if (!ws && wsStatus !== "offline") {
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
        return (
          <ContactsPage
            setSelectedContact={setSelectedContact}
            onlineUsers={onlineUsers}
            invites={invites}
            onInvitesUpdate={fetchInvites}
          />
        );
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
            <div className="nav-icon-wrapper">
              <PersonIcon />
              {invites.length > 0 && <div className="notification-dot"></div>}
            </div>
            <span>People</span>
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
