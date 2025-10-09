import Contacts from "./Contacts";

export default function ContactsPage({ setSelectedContact }) {
  return (
    <div className="contacts-page">
      <Contacts setSelectedContact={setSelectedContact} />
    </div>
  );
}
