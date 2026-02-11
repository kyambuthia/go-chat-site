import { useState, useEffect, useMemo } from "react";
import { getContacts } from "../api";
import { CheckIcon, PaperPlaneIcon, ArrowUpIcon, PersonIcon, MagnifyingGlassIcon } from "@radix-ui/react-icons";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";
import SendMoneyForm from "./SendMoneyForm";

function ChatWindow({ ws, selectedContact, messages, onSendMessage, onBack, isOnline }) {
  const [newMessage, setNewMessage] = useState("");
  const [showSendMoneyForm, setShowSendMoneyForm] = useState(false);

  const handleSendMessage = () => {
    const trimmed = newMessage.trim();
    if (!trimmed || !selectedContact) {
      return;
    }

    const message = {
      id: Date.now(),
      type: "direct_message",
      to: selectedContact.username,
      body: trimmed,
    };

    ws.send(JSON.stringify(message));
    onSendMessage(message);
    setNewMessage("");
  };

  if (!selectedContact) {
    return <div className="chat-window placeholder">Select a contact to start chatting.</div>;
  }

  return (
    <div className="chat-window">
      <div className="chat-header">
        <button onClick={onBack} className="back-button">←</button>
        <Avatar className={`avatar-placeholder ${isOnline ? "online" : ""}`}>
          <AvatarImage src="" alt="" />
          <AvatarFallback><PersonIcon width="24" height="24" /></AvatarFallback>
        </Avatar>
        <h2>{selectedContact.display_name || selectedContact.username}</h2>
        <button onClick={() => setShowSendMoneyForm(true)} className="send-money-chat-button">
          <ArrowUpIcon />
        </button>
      </div>
      {showSendMoneyForm ? (
        <SendMoneyForm
          defaultRecipient={selectedContact.username}
          onSendSuccess={() => setShowSendMoneyForm(false)}
          onCancel={() => setShowSendMoneyForm(false)}
        />
      ) : (
        <>
          <div className="messages">
            {messages.length === 0 && <p className="thread-empty">No messages yet. Start the conversation.</p>}
            {messages.map((msg, index) => (
              <div key={index} className={`message ${msg.sent ? "sent" : "received"}`}>
                <div className="message-body">{msg.body}</div>
                {msg.sent && msg.delivered && <span className="message-status-icon"><CheckIcon /></span>}
              </div>
            ))}
          </div>
          <div className="input-area">
            <input
              type="text"
              value={newMessage}
              onChange={(e) => setNewMessage(e.target.value)}
              placeholder="Type a message..."
              onKeyUp={(e) => e.key === "Enter" && handleSendMessage()}
            />
            <button onClick={handleSendMessage} className="send-button"><PaperPlaneIcon className="send-button-icon" /></button>
          </div>
        </>
      )}
    </div>
  );
}

export default function Chat({ ws, selectedContact, setSelectedContact, onlineUsers, lastWsMessage }) {
  const [threads, setThreads] = useState({});
  const [unreadByUser, setUnreadByUser] = useState({});
  const [contacts, setContacts] = useState([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const selectedUsername = selectedContact?.username;
  const messages = selectedUsername ? (threads[selectedUsername] || []) : [];

  useEffect(() => {
    const fetchContacts = async () => {
      try {
        setLoading(true);
        const response = await getContacts();
        setContacts(response || []);
      } catch (err) {
        console.error("Failed to fetch contacts:", err);
        setError(err.message);
        setContacts([]);
      } finally {
        setLoading(false);
      }
    };
    fetchContacts();
  }, []);

  useEffect(() => {
    if (!lastWsMessage) {
      return;
    }

    if (lastWsMessage.type === "message_ack") {
      setThreads((prev) => {
        const next = { ...prev };
        Object.keys(next).forEach((username) => {
          next[username] = next[username].map((msg) =>
            msg.id === lastWsMessage.id ? { ...msg, delivered: true } : msg
          );
        });
        return next;
      });
      return;
    }

    if (lastWsMessage.type === "direct_message" && lastWsMessage.from) {
      setThreads((prev) => ({
        ...prev,
        [lastWsMessage.from]: [...(prev[lastWsMessage.from] || []), lastWsMessage],
      }));

      if (selectedUsername !== lastWsMessage.from) {
        setUnreadByUser((prev) => ({
          ...prev,
          [lastWsMessage.from]: (prev[lastWsMessage.from] || 0) + 1,
        }));
      }
    }
  }, [lastWsMessage, selectedUsername]);

  useEffect(() => {
    if (!selectedUsername) {
      return;
    }
    setUnreadByUser((prev) => ({ ...prev, [selectedUsername]: 0 }));
  }, [selectedUsername]);

  const filteredContacts = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const base = contacts.filter((contact) => {
      if (!needle) {
        return true;
      }
      const display = (contact.display_name || "").toLowerCase();
      return contact.username.toLowerCase().includes(needle) || display.includes(needle);
    });

    return base.sort((a, b) => {
      const aUnread = unreadByUser[a.username] || 0;
      const bUnread = unreadByUser[b.username] || 0;
      if (aUnread !== bUnread) {
        return bUnread - aUnread;
      }
      const aOnline = onlineUsers.includes(a.username) ? 1 : 0;
      const bOnline = onlineUsers.includes(b.username) ? 1 : 0;
      if (aOnline !== bOnline) {
        return bOnline - aOnline;
      }
      return a.username.localeCompare(b.username);
    });
  }, [contacts, onlineUsers, query, unreadByUser]);

  const handleSelectContact = (contact) => {
    setSelectedContact(contact);
  };

  const handleSendMessage = (message) => {
    if (!selectedUsername) {
      return;
    }

    setThreads((prev) => ({
      ...prev,
      [selectedUsername]: [...(prev[selectedUsername] || []), { ...message, from: "Me", sent: true, delivered: false }],
    }));
  };

  if (loading) {
    return (
      <div className="chat-main">
        <h2>Chat</h2>
        <p>Loading contacts...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="chat-main">
        <h2>Chat</h2>
        <p>Error: {error}</p>
      </div>
    );
  }

  if (!selectedContact) {
    return (
      <div className="chat-main">
        <h2>Chat</h2>
        {contacts.length === 0 ? (
          <p className="empty-chat-message">Invite a friend from the Contacts tab to start chatting.</p>
        ) : (
          <div className="contacts-list">
            <div className="contacts-toolbar">
              <div className="search-wrap">
                <MagnifyingGlassIcon />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Search contacts"
                  aria-label="Search contacts"
                />
              </div>
            </div>
            <ul>
              {filteredContacts.map((contact) => {
                const unread = unreadByUser[contact.username] || 0;
                return (
                  <li key={contact.id} onClick={() => handleSelectContact(contact)}>
                    <Avatar className={`avatar-placeholder ${onlineUsers.includes(contact.username) ? "online" : ""}`}>
                      <AvatarImage src="" alt="" />
                      <AvatarFallback><PersonIcon width="24" height="24" /></AvatarFallback>
                    </Avatar>
                    <div className="contact-meta">
                      <span>{contact.display_name || contact.username}</span>
                      <small>{onlineUsers.includes(contact.username) ? "Online" : "Offline"}</small>
                    </div>
                    {unread > 0 && <span className="unread-pill">{unread}</span>}
                  </li>
                );
              })}
            </ul>
          </div>
        )}
      </div>
    );
  }

  return (
    <ChatWindow
      ws={ws}
      selectedContact={selectedContact}
      messages={messages}
      onSendMessage={handleSendMessage}
      onBack={() => setSelectedContact(null)}
      isOnline={onlineUsers.includes(selectedContact.username)}
    />
  );
}
