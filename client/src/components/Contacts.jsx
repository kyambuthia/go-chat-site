import { useState, useEffect, useMemo } from "react";
import { getContacts } from "../api";
import Invite from "./Invite";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";
import { PersonIcon, MagnifyingGlassIcon } from "@radix-ui/react-icons";

export default function Contacts({ setSelectedContact, onlineUsers }) {
  const [contacts, setContacts] = useState([]);
  const [query, setQuery] = useState("");
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

  const filteredContacts = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) {
      return contacts;
    }
    return contacts.filter((contact) => {
      const display = (contact.display_name || "").toLowerCase();
      return contact.username.toLowerCase().includes(needle) || display.includes(needle);
    });
  }, [contacts, query]);

  if (loading) {
    return <div className="contacts-list">Loading contacts...</div>;
  }

  if (error) {
    return <div className="contacts-list">Error: {error}</div>;
  }

  return (
    <div className="contacts-list">
      <h2>My Contacts</h2>
      {contacts.length === 0 ? (
        <div className="empty-contacts">
          <h3>No Contacts Yet</h3>
          <p>Send an invite to start a conversation.</p>
          <Invite />
        </div>
      ) : (
        <>
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
          {filteredContacts.length === 0 ? (
            <p className="empty-chat-message">No contacts match your search.</p>
          ) : (
            <ul>
              {filteredContacts.map((contact) => (
                <li key={contact.id} onClick={() => setSelectedContact(contact)}>
                  <Avatar className={`avatar-placeholder ${onlineUsers.includes(contact.username) ? "online" : ""}`}>
                    <AvatarImage src="" alt="" />
                    <AvatarFallback><PersonIcon width="24" height="24" /></AvatarFallback>
                  </Avatar>
                  <div className="contact-meta">
                    <span>{contact.display_name || contact.username}</span>
                    <small>{onlineUsers.includes(contact.username) ? "Online" : "Offline"}</small>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </div>
  );
}
