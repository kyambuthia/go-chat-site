import { acceptInvite, rejectInvite } from "../api";

export default function InvitesPage({ invites, onUpdate }) {
  const handleAccept = async (inviteID) => {
    await acceptInvite(inviteID);
    onUpdate();
  };

  const handleReject = async (inviteID) => {
    await rejectInvite(inviteID);
    onUpdate();
  };

  return (
    <div className="invites-page">
      <h2>Pending Invites</h2>
      {invites.length === 0 ? (
        <p>No pending invites.</p>
      ) : (
        <ul>
          {invites.map((invite) => (
            <li key={invite.id}>
              <div className="avatar-placeholder">{invite.inviter_username.charAt(0).toUpperCase()}</div>
              <span>{invite.inviter_username}</span>
              <div className="invite-actions">
                <button onClick={() => handleAccept(invite.id)} className="accept">Accept</button>
                <button onClick={() => handleReject(invite.id)} className="reject">Reject</button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
