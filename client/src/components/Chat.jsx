import { useState, useEffect } from "react";
import { getContacts } from "../api";
import { CheckIcon, PaperPlaneIcon, ArrowUpIcon, PersonIcon } from "@radix-ui/react-icons";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";
import SendMoneyForm from "./SendMoneyForm";

function ChatWindow({ ws, selectedContact, messages, setMessages, onBack, isOnline }) {
  const [newMessage, setNewMessage] = useState("");
  const [showSendMoneyForm, setShowSendMoneyForm] = useState(false);

  const handleSendMessage = () => {
    if (!newMessage.trim() || !selectedContact) {
      return;
    }

    const message = {
      id: Date.now(),
      type: "direct_message",
      to: selectedContact.username,
      body: newMessage,
    };

    ws.send(JSON.stringify(message));
    setMessages((prev) => [...prev, { ...message, from: "Me", sent: true, delivered: false }]);
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
  const [messages, setMessages] = useState([]);
  const [contacts, setContacts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

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
      setMessages((prevMessages) =>
        prevMessages.map((msg) => (msg.id === lastWsMessage.id ? { ...msg, delivered: true } : msg))
      );
      return;
    }

    if (lastWsMessage.type === "direct_message") {
      setMessages((prevMessages) => [...prevMessages, lastWsMessage]);
    }
  }, [lastWsMessage]);

  useEffect(() => {
    setMessages([]);
  }, [selectedContact]);

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
            <h3>Your Contacts</h3>
            <ul>
              {contacts.map((contact) => (
                <li key={contact.id} onClick={() => setSelectedContact(contact)}>
                  <Avatar className={`avatar-placeholder ${onlineUsers.includes(contact.username) ? "online" : ""}`}>
                    <AvatarImage src="" alt="" />
                    <AvatarFallback><PersonIcon width="24" height="24" /></AvatarFallback>
                  </Avatar>
                  <span>{contact.display_name || contact.username}</span>
                </li>
              ))}
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
      setMessages={setMessages}
      onBack={() => setSelectedContact(null)}
      isOnline={onlineUsers.includes(selectedContact.username)}
    />
  );
}
