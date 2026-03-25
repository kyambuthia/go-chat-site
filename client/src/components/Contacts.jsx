import { useState, useEffect, useMemo } from "react";
import { addContact, getContacts, removeContact } from "../api";
import Invite from "./Invite";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";
import { PersonIcon, MagnifyingGlassIcon } from "@radix-ui/react-icons";

export default function Contacts({ setSelectedContact, onlineUsers }) {
  const [contacts, setContacts] = useState([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [actionStatus, setActionStatus] = useState({ type: "", message: "" });
  const [removingContactID, setRemovingContactID] = useState(null);
  const [newContactUsername, setNewContactUsername] = useState("");
  const [addingContact, setAddingContact] = useState(false);

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
    setActionStatus({ type: "", message: "" });
    setRemovingContactID(contactID);

    try {
      await removeContact(contactID);
      setContacts((currentContacts) => currentContacts.filter((contact) => contact.id !== contactID));
      setActionStatus({ type: "success", message: "Contact removed." });
    } catch (err) {
      console.error("Failed to remove contact:", err);
      setActionStatus({ type: "error", message: err.message || "Could not remove contact." });
    } finally {
      setRemovingContactID(null);
    }
  };

  const handleAddContact = async (event) => {
    event.preventDefault();
    const username = newContactUsername.trim();
    if (!username) {
      return;
    }

    setActionStatus({ type: "", message: "" });
    setAddingContact(true);

    try {
      await addContact(username);
      const refreshedContacts = await getContacts();
      setContacts(refreshedContacts || []);
      setNewContactUsername("");
      setQuery("");
      setActionStatus({ type: "success", message: `Added ${username} to contacts.` });
    } catch (err) {
      console.error("Failed to add contact:", err);
      setActionStatus({ type: "error", message: err.message || "Could not add contact." });
    } finally {
      setAddingContact(false);
    }
  };

  return (
    <div className="contacts-list">
      <h2>My Contacts</h2>
      <div className="contacts-actions">
        <div className="contacts-action-grid">
          <form className="contacts-inline-form" onSubmit={handleAddContact}>
            <div className="form-group">
              <label htmlFor="add-contact-username">Add contact directly</label>
              <input
                id="add-contact-username"
                type="text"
                value={newContactUsername}
                onChange={(event) => setNewContactUsername(event.target.value)}
                placeholder="username"
                disabled={addingContact}
                required
              />
            </div>
            <button type="submit" disabled={addingContact || !newContactUsername.trim()}>
              {addingContact ? "Adding..." : "Add Contact"}
            </button>
          </form>
          <Invite compact />
          {actionStatus.message && (
            <p className={`money-status contacts-status ${actionStatus.type === "error" ? "is-error" : "is-success"}`}>
              {actionStatus.message}
            </p>
          )}
        </div>
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
