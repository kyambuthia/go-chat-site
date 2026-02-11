import { useEffect, useState } from "react";
import { acceptInvite, rejectInvite } from "../api";
import { PersonIcon } from "@radix-ui/react-icons";
import { Avatar, AvatarFallback, AvatarImage } from "@radix-ui/react-avatar";

export default function InvitesPage({ invites, onUpdate }) {
  const [localInvites, setLocalInvites] = useState(invites);
  const [actioningInvite, setActioningInvite] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    setLocalInvites(invites);
  }, [invites]);

  const handleInviteAction = async (inviteID, action) => {
    setError("");
    setActioningInvite(inviteID);

    const previous = localInvites;
    setLocalInvites((curr) => curr.filter((invite) => invite.id !== inviteID));

    try {
      if (action === "accept") {
        await acceptInvite(inviteID);
      } else {
        await rejectInvite(inviteID);
      }
      await onUpdate();
    } catch (err) {
      setLocalInvites(previous);
      setError(err.message || "Could not update invite. Please try again.");
    } finally {
      setActioningInvite(null);
    }
  };

  return (
    <div className="invites-page">
      <h2>Pending Invites</h2>
      {error && <p className="error-message invite-error">{error}</p>}
      {localInvites.length === 0 ? (
        <p>No pending invites.</p>
      ) : (
        <ul>
          {localInvites.map((invite) => {
            const isBusy = actioningInvite === invite.id;
            return (
              <li key={invite.id}>
                <Avatar className="avatar-placeholder">
                  <AvatarImage src="" alt={invite.inviter_username} />
                  <AvatarFallback><PersonIcon width="24" height="24" /></AvatarFallback>
                </Avatar>
                <span>{invite.inviter_username}</span>
                <div className="invite-actions">
                  <button onClick={() => handleInviteAction(invite.id, "accept")} className="accept" disabled={isBusy}>
                    {isBusy ? "Working..." : "Accept"}
                  </button>
                  <button onClick={() => handleInviteAction(invite.id, "reject")} className="reject" disabled={isBusy}>
                    {isBusy ? "Working..." : "Reject"}
                  </button>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
