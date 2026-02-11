import Contacts from "./Contacts";
import InvitesPage from "./InvitesPage";

export default function ContactsPage({ setSelectedContact, onlineUsers, invites, onInvitesUpdate }) {
  return (
    <div className="contacts-page combined-tab">
      <section className="combined-section">
        <Contacts setSelectedContact={setSelectedContact} onlineUsers={onlineUsers} />
      </section>
      <section className="combined-section">
        <InvitesPage invites={invites} onUpdate={onInvitesUpdate} compact />
      </section>
    </div>
  );
}
