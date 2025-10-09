import { useState, useEffect } from "react";
import { getContacts } from "../api";
import Invite from "./Invite";

export default function Contacts({ setSelectedContact }) {
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

  if (loading) {
    return <div className="contacts-list">Loading contacts...</div>;
  }

  if (error) {
    return <div className="contacts-list">Error: {error}</div>;
  }

  if (contacts.length === 0) {
    return (
      <div className="contacts-list">
        <div className="empty-contacts">
          <h2>No Contacts Yet</h2>
          <p>Send an invite to start a conversation.</p>
          <Invite />
        </div>
      </div>
    );
  }

  return (
    <div className="contacts-list">
      <h2>My Contacts</h2>
      <ul>
        {contacts.map((contact) => (
          <li key={contact.id} onClick={() => setSelectedContact(contact)}>
            {/* <img src={contact.avatar_url} alt={contact.display_name} /> */}
            <span>{contact.display_name || contact.username}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}