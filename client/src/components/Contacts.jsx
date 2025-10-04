import { useState, useEffect } from "react";
import { getContacts, addContact } from "../api";

export default function Contacts({ setSelectedContact }) {
  const [contacts, setContacts] = useState([]);
  const [newContactUsername, setNewContactUsername] = useState("");

  useEffect(() => {
    const fetchContacts = async () => {
      const response = await getContacts();
              setContacts(response || []);    };
    fetchContacts();
  }, []);

  const handleAddContact = async (e) => {
    e.preventDefault();
    if (newContactUsername.trim()) {
      try {
        await addContact(newContactUsername);
        setNewContactUsername("");
        // Refresh contacts list
        const response = await getContacts();
      setContacts(response || []);
      } catch (error) {
        console.error("Error adding contact:", error);
        // You might want to show an error to the user here
      }
    }
  };

  return (
    <div className="contacts-list">
      <div className="add-contact">
        <form onSubmit={handleAddContact}>
          <input
            type="text"
            value={newContactUsername}
            onChange={(e) => setNewContactUsername(e.target.value)}
            placeholder="Add contact by username"
          />
          <button type="submit">Add</button>
        </form>
      </div>
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