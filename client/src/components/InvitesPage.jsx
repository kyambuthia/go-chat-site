import { useState, useEffect } from "react";
import { getInvites, acceptInvite, rejectInvite } from "../api";

export default function InvitesPage() {
  const [invites, setInvites] = useState([]);

  useEffect(() => {
    const fetchInvites = async () => {
      const response = await getInvites();
      setInvites(response || []);
    };
    fetchInvites();
  }, []);

  const handleAccept = async (inviteID) => {
    await acceptInvite(inviteID);
    setInvites(invites.filter((invite) => invite.id !== inviteID));
  };

  const handleReject = async (inviteID) => {
    await rejectInvite(inviteID);
    setInvites(invites.filter((invite) => invite.id !== inviteID));
  };

  return (
    <div className="invites-page">
      <h2>Pending Invites</h2>
      <ul>
        {invites.map((invite) => (
          <li key={invite.id}>
            <span>{invite.inviter_username}</span>
            <button onClick={() => handleAccept(invite.id)}>Accept</button>
            <button onClick={() => handleReject(invite.id)}>Reject</button>
          </li>
        ))}
      </ul>
    </div>
  );
}
