import Contacts from "./Contacts";
import Invite from "./Invite";

export default function ContactsPage({ setSelectedContact }) {
  return (
    <div className="contacts-page">
      <Contacts setSelectedContact={setSelectedContact} />
      <Invite />
    </div>
  );
}
