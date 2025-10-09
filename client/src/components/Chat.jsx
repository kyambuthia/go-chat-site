import { useState, useEffect } from "react";
import { getContacts } from "../api";

function ChatWindow({ ws, selectedContact, messages, setMessages, onBack }) {
  const [newMessage, setNewMessage] = useState("");

  // TODO: Fetch message history with selectedContact when it changes
  // useEffect(() => {
  //   if (selectedContact) {
  //     getMessages(selectedContact.ID).then(response => {
  //       setMessages(response.data);
  //     });
  //   }
  // }, [selectedContact]);

  const handleSendMessage = () => {
    if (newMessage.trim() && selectedContact) {
      const message = {
        id: Date.now(), // Simple unique ID
        type: "direct_message",
        to: selectedContact.username, // Assuming the API uses Username
        body: newMessage,
      };
      ws.send(JSON.stringify(message));
      // Add the message to the local state to display it immediately
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
        <button onClick={onBack}>Back</button>
        <img src={selectedContact.avatar_url} alt={selectedContact.display_name} />
        <h2>{selectedContact.display_name}</h2>
      </div>
      <div className="messages">
        {messages.map((msg, index) => (
          <div key={index} className={`message ${msg.sent ? 'sent' : 'received'}`}>
            {msg.body}
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

  useEffect(() => {
    const fetchContacts = async () => {
      const response = await getContacts();
      setContacts(response || []);
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

  if (!selectedContact) {
    return (
      <div className="contacts-list">
        <h2>Contacts</h2>
        <ul>
          {contacts.map((contact) => (
            <li key={contact.id} onClick={() => setSelectedContact(contact)}>
              {contact.username}
            </li>
          ))}
        </ul>
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