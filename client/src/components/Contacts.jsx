import { useState, useEffect } from "react";
import { getContacts } from "../api";

export default function Contacts({ setSelectedContact }) {
  const [contacts, setContacts] = useState([]);

  useEffect(() => {
    const fetchContacts = async () => {
      const response = await getContacts();
      setContacts(response || []);
    };
    fetchContacts();
  }, []);

  return (
    <div className="contacts-list">
      <ul>
        {contacts.map((contact) => (
          <li key={contact.id} onClick={() => setSelectedContact(contact)}>
            <img src={contact.avatar_url} alt={contact.display_name} />
            <span>{contact.display_name}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}