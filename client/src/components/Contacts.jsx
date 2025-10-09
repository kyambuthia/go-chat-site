import { useState, useEffect } from "react";
import { getContacts } from "../api";
import Invite from "./Invite";

export default function Contacts({ setSelectedContact }) {
  const [contacts, setContacts] = useState([]);

  useEffect(() => {
    const fetchContacts = async () => {
      try {
        const response = await getContacts();
        setContacts(response || []);
      } catch (error) {
        console.error("Failed to fetch contacts:", error);
        setContacts([]);
      }
    };
    fetchContacts();
  }, []);

  if (contacts.length === 0) {
    return (
      <div className="empty-contacts">
        <h2>No Contacts Yet</h2>
        <p>Send an invite to start a conversation.</p>
        <Invite />
      </div>
    );
  }

  return (
    <div className="contacts-list">
      <h2>Contacts</h2>
      <ul>
        {contacts.map((contact) => (
          <li key={contact.id} onClick={() => setSelectedContact(contact)}>
            <img src={contact.avatar_url} alt={contact.display_name} />
            <span>{contact.display_name || contact.username}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
