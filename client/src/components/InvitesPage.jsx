import { acceptInvite, rejectInvite } from "../api";
import { PersonIcon } from '@radix-ui/react-icons';
import { Avatar, AvatarFallback, AvatarImage } from '@radix-ui/react-avatar';

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
              <Avatar className="avatar-placeholder">
                <AvatarImage src="" alt={invite.inviter_username} />
                <AvatarFallback><PersonIcon width="24" height="24" /></AvatarFallback>
              </Avatar>
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
