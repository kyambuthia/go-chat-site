import Contacts from "./Contacts";

export default function ContactsPage({ setSelectedContact, onlineUsers }) {
  return (
    <div className="contacts-page">
      <Contacts setSelectedContact={setSelectedContact} onlineUsers={onlineUsers} />
    </div>
  );
}
