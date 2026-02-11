import { useState } from "react";
import Contacts from "./Contacts";
import InvitesPage from "./InvitesPage";

export default function ContactsPage({ setSelectedContact, onlineUsers, invites, onInvitesUpdate }) {
  const [view, setView] = useState("contacts");

  return (
    <div className="contacts-page combined-tab segmented-people">
      <div className="segment-switch" role="tablist" aria-label="People sections">
        <button
          type="button"
          role="tab"
          aria-selected={view === "contacts"}
          className={`segment-btn ${view === "contacts" ? "active" : ""}`}
          onClick={() => setView("contacts")}
        >
          Contacts
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={view === "invites"}
          className={`segment-btn ${view === "invites" ? "active" : ""}`}
          onClick={() => setView("invites")}
        >
          Invites {invites.length > 0 ? `(${invites.length})` : ""}
        </button>
      </div>

      <section className="combined-section">
        {view === "contacts" ? (
          <Contacts setSelectedContact={setSelectedContact} onlineUsers={onlineUsers} />
        ) : (
          <InvitesPage invites={invites} onUpdate={onInvitesUpdate} compact />
        )}
      </section>
    </div>
  );
}
