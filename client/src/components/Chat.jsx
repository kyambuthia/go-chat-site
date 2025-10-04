import { useState, useEffect } from "react";
import Contacts from "./Contacts";

function ChatWindow({ ws, selectedContact, messages, setMessages }) {
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
        type: "direct_message",
        to: selectedContact.username, // Assuming the API uses Username
        body: newMessage,
      };
      ws.send(JSON.stringify(message));
      // Add the message to the local state to display it immediately
      setMessages([...messages, { from: "Me", body: newMessage }]);
      setNewMessage("");
    }
  };

  if (!selectedContact) {
    return <div className="chat-window placeholder">Select a contact to start chatting.</div>;
  }

  return (
    <div className="chat-window">
      <div className="chat-header">{selectedContact.username}</div>
      <div className="messages">
        {messages.map((msg, index) => (
          <div key={index} className={`message ${msg.from === 'Me' ? 'sent' : 'received'}`}>
            <b>{msg.from || selectedContact.username}:</b> {msg.body}
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

export default function Chat({ ws }) {
  const [selectedContact, setSelectedContact] = useState(null);
  const [messages, setMessages] = useState([]);

  useEffect(() => {
    ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      // We need to make sure we're showing the message in the right context
      // This simple logic assumes the message is from the selected contact
      setMessages((prevMessages) => [...prevMessages, message]);
    };
  }, [ws, selectedContact]); // Re-bind if contact changes

  // Clear messages when contact changes
  useEffect(() => {
    setMessages([]);
  }, [selectedContact]);

  return (
    <div className="chat-container">
      <Contacts setSelectedContact={setSelectedContact} />
      <ChatWindow 
        ws={ws} 
        selectedContact={selectedContact} 
        messages={messages} 
        setMessages={setMessages} 
      />
    </div>
  );
}