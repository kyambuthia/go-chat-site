import { useState, useEffect } from "react";
import { getContacts } from "../api";

function ChatWindow({ ws, selectedContact, messages, setMessages, onBack }) {
  const [newMessage, setNewMessage] = useState("");

  const handleSendMessage = () => {
    if (newMessage.trim() && selectedContact) {
      const message = {
        id: Date.now(),
        type: "direct_message",
        to: selectedContact.username,
        body: newMessage,
      };
      ws.send(JSON.stringify(message));
      setMessages([...messages, { ...message, from: "Me", sent: true, delivered: false }]);
      setNewMessage("");
    }
  };

  if (!selectedContact) {
    return <div className="chat-window placeholder">Select a contact to start chatting.</div>;
  }

  return (
    <div className="chat-window">
      <div className="chat-header">
        <button onClick={onBack} className="back-button">←</button>
        <div className="avatar-placeholder">{selectedContact.username.charAt(0).toUpperCase()}</div>
        <h2>{selectedContact.display_name || selectedContact.username}</h2>
      </div>
      <div className="messages">
        {messages.map((msg, index) => (
          <div key={index} className={`message ${msg.sent ? 'sent' : 'received'}`}>
            <div className="message-body">{msg.body}</div>
            {msg.sent && <span className="tick">{msg.delivered ? '✔' : ''}</span>}
          </div>
        ))}
      </div>
      <div className="input-area">
        <input
          type="text"
          value={newMessage}
          onChange={(e) => setNewMessage(e.target.value)}
          placeholder="Type a message..."
          onKeyPress={(e) => e.key === 'Enter' && handleSendMessage()}
        />
        <button onClick={handleSendMessage}>Send</button>
      </div>
    </div>
  );
}

export default function Chat({ ws, selectedContact, setSelectedContact }) {
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
    if (selectedContact) {
      ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        if (message.type === "message_ack") {
          setMessages((prevMessages) =>
            prevMessages.map((msg) =>
              msg.id === message.id ? { ...msg, delivered: true } : msg
            )
          );
        } else {
          setMessages((prevMessages) => [...prevMessages, message]);
        }
      };
    }
  }, [ws, selectedContact]);

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
          <div className="contacts-list-small">
            <h3>Your Contacts</h3>
            <ul>
              {contacts.map((contact) => (
                <li key={contact.id} onClick={() => setSelectedContact(contact)}>
                  <div className="avatar-placeholder">{contact.username.charAt(0).toUpperCase()}</div>
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
    />
  );
}
