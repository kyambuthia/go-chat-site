package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func (s *SqliteStore) ListContacts(userID int) ([]User, error) {
	rows, err := s.DB.Query(`
		SELECT u.id, u.username, COALESCE(u.display_name, ''), COALESCE(u.avatar_url, '')
		FROM users u
		INNER JOIN contacts c ON u.id = c.contact_id
		WHERE c.user_id = ?
		ORDER BY COALESCE(u.display_name, u.username), u.username
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contacts := make([]User, 0)
	for rows.Next() {
		var contact User
		if err := rows.Scan(&contact.ID, &contact.Username, &contact.DisplayName, &contact.AvatarURL); err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return contacts, nil
}

func (s *SqliteStore) AddContact(userID, contactID int) error {
	if userID == contactID {
		return errors.New("you cannot add yourself as a contact")
	}
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO contacts (user_id, contact_id) VALUES (?, ?)`, userID, contactID)
	return err
}

func (s *SqliteStore) RemoveContact(userID, contactID int) error {
	_, err := s.DB.Exec(`DELETE FROM contacts WHERE user_id = ? AND contact_id = ?`, userID, contactID)
	return err
}

func (s *SqliteStore) CreateInvite(inviterID, inviteeID int) error {
	if inviterID == inviteeID {
		return errors.New("you cannot invite yourself")
	}

	_, err := s.DB.Exec(`
		INSERT INTO contact_invites (requester_id, recipient_id, status)
		VALUES (?, ?, 'pending')
	`, inviterID, inviteeID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return ErrInviteExists
		}
		return err
	}
	return nil
}

func (s *SqliteStore) ListInvites(userID int) ([]Invite, error) {
	rows, err := s.DB.Query(`
		SELECT ci.id, u.username
		FROM contact_invites ci
		INNER JOIN users u ON ci.requester_id = u.id
		WHERE ci.recipient_id = ? AND ci.status = 'pending'
		ORDER BY ci.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invites := make([]Invite, 0)
	for rows.Next() {
		var invite Invite
		if err := rows.Scan(&invite.ID, &invite.InviterUsername); err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invites, nil
}

func (s *SqliteStore) UpdateInviteStatus(inviteID, userID int, status string) error {
	if status != "accepted" && status != "rejected" {
		return fmt.Errorf("unsupported status: %s", status)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var inviterID int
	row := tx.QueryRow(`
		SELECT requester_id
		FROM contact_invites
		WHERE id = ? AND recipient_id = ? AND status = 'pending'
	`, inviteID, userID)

	if err := row.Scan(&inviterID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	_, err = tx.Exec(`
		UPDATE contact_invites
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, inviteID)
	if err != nil {
		return err
	}

	if status == "accepted" {
		_, err = tx.Exec(`
			INSERT OR IGNORE INTO contacts (user_id, contact_id)
			VALUES (?, ?), (?, ?)
		`, userID, inviterID, inviterID, userID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
