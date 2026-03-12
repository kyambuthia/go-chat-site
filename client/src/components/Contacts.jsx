import { useState, useEffect, useMemo } from "react";
import { getContacts, removeContact } from "../api";
import Invite from "./Invite";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";
import { PersonIcon, MagnifyingGlassIcon } from "@radix-ui/react-icons";

export default function Contacts({ setSelectedContact, onlineUsers }) {
  const [contacts, setContacts] = useState([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [actionError, setActionError] = useState(null);
  const [removingContactID, setRemovingContactID] = useState(null);

  useEffect(() => {
    const loadContacts = async () => {
      try {
        setLoading(true);
        setError(null);
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

    loadContacts();
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

  const handleRemoveContact = async (event, contactID) => {
    event.stopPropagation();
    setActionError(null);
    setRemovingContactID(contactID);

    try {
      await removeContact(contactID);
      setContacts((currentContacts) => currentContacts.filter((contact) => contact.id !== contactID));
    } catch (err) {
      console.error("Failed to remove contact:", err);
      setActionError(err.message || "Could not remove contact.");
    } finally {
      setRemovingContactID(null);
    }
  };

  return (
    <div className="contacts-list">
      <h2>My Contacts</h2>
      <div className="contacts-actions">
        <Invite compact />
        {actionError && <p className="error-message contacts-error">{actionError}</p>}
      </div>
      {contacts.length === 0 ? (
        <div className="empty-contacts">
          <h3>No Contacts Yet</h3>
          <p>Send an invite to start a conversation.</p>
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
                    <AvatarImage src={contact.avatar_url || ""} alt={contact.display_name || contact.username} />
                    <AvatarFallback><PersonIcon width="24" height="24" /></AvatarFallback>
                  </Avatar>
                  <div className="contact-meta">
                    <span>{contact.display_name || contact.username}</span>
                    <small>{onlineUsers.includes(contact.username) ? "Online" : "Offline"}</small>
                  </div>
                  <div className="contact-item-actions">
                    <button
                      type="button"
                      className="contact-action-btn danger"
                      onClick={(event) => handleRemoveContact(event, contact.id)}
                      disabled={removingContactID === contact.id}
                    >
                      {removingContactID === contact.id ? "Removing..." : "Remove"}
                    </button>
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
