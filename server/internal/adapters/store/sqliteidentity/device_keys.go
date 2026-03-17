package sqliteidentity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type DeviceKeysAdapter struct {
	DB *sql.DB
}

var _ coreid.DeviceIdentityRepository = (*DeviceKeysAdapter)(nil)

func (a *DeviceKeysAdapter) CreateDeviceIdentity(ctx context.Context, userID coreid.UserID, sessionID int64, req coreid.RegisterDeviceIdentityRequest) (coreid.DeviceIdentity, error) {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return coreid.DeviceIdentity{}, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `
		INSERT INTO device_identities (
			user_id, label, algorithm, identity_key, signed_prekey_id,
			signed_prekey, signed_prekey_signature, key_state,
			created_at, published_at, rotated_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, req.Label, req.Algorithm, req.IdentityKey, req.SignedPrekeyID, req.SignedPrekey, req.SignedPrekeySignature, coreid.DeviceKeyStateActive, now, now, now, now)
	if err != nil {
		return coreid.DeviceIdentity{}, err
	}
	deviceID, err := result.LastInsertId()
	if err != nil {
		return coreid.DeviceIdentity{}, err
	}

	if err := upsertDeviceSessionTx(ctx, tx, deviceID, sessionID, now); err != nil {
		return coreid.DeviceIdentity{}, err
	}
	if err := upsertPrekeysTx(ctx, tx, deviceID, req.Prekeys, now); err != nil {
		return coreid.DeviceIdentity{}, err
	}

	if err := tx.Commit(); err != nil {
		return coreid.DeviceIdentity{}, err
	}
	return a.getDeviceIdentity(ctx, userID, sessionID, deviceID)
}

func (a *DeviceKeysAdapter) ListDeviceIdentities(ctx context.Context, userID coreid.UserID, sessionID int64) ([]coreid.DeviceIdentity, error) {
	rows, err := a.DB.QueryContext(ctx, `
		SELECT
			di.id,
			di.user_id,
			di.label,
			di.algorithm,
			di.identity_key,
			di.signed_prekey_id,
			di.signed_prekey,
			di.signed_prekey_signature,
			di.key_state,
			di.created_at,
			di.published_at,
			di.rotated_at,
			di.revoked_at,
			COALESCE(COUNT(CASE WHEN dp.key_state = 'active' AND dp.revoked_at IS NULL THEN 1 END), 0) AS prekey_count,
			MAX(CASE WHEN ds.auth_session_id = ? THEN 1 ELSE 0 END) AS current_session
		FROM device_identities di
		LEFT JOIN device_prekeys dp ON dp.device_identity_id = di.id
		LEFT JOIN device_sessions ds ON ds.device_identity_id = di.id
		WHERE di.user_id = ?
		GROUP BY di.id
		ORDER BY di.created_at DESC
	`, sessionID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]coreid.DeviceIdentity, 0)
	for rows.Next() {
		device, err := scanDeviceIdentity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, device)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *DeviceKeysAdapter) RotateDeviceIdentity(ctx context.Context, userID coreid.UserID, sessionID int64, req coreid.RotateDeviceIdentityRequest) (coreid.DeviceIdentity, error) {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return coreid.DeviceIdentity{}, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `
		UPDATE device_identities
		SET signed_prekey_id = ?, signed_prekey = ?, signed_prekey_signature = ?, rotated_at = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND revoked_at IS NULL
	`, req.SignedPrekeyID, req.SignedPrekey, req.SignedPrekeySignature, now, now, req.DeviceID, userID)
	if err != nil {
		return coreid.DeviceIdentity{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return coreid.DeviceIdentity{}, err
	}
	if rowsAffected == 0 {
		return coreid.DeviceIdentity{}, coreid.ErrDeviceIdentityNotFound
	}

	if err := upsertDeviceSessionTx(ctx, tx, req.DeviceID, sessionID, now); err != nil {
		return coreid.DeviceIdentity{}, err
	}
	if len(req.Prekeys) > 0 {
		if err := upsertPrekeysTx(ctx, tx, req.DeviceID, req.Prekeys, now); err != nil {
			return coreid.DeviceIdentity{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return coreid.DeviceIdentity{}, err
	}
	return a.getDeviceIdentity(ctx, userID, sessionID, req.DeviceID)
}

func (a *DeviceKeysAdapter) PublishPrekeys(ctx context.Context, userID coreid.UserID, req coreid.PublishPrekeysRequest) ([]coreid.DevicePrekey, error) {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ok, err := deviceBelongsToUserTx(ctx, tx, req.DeviceID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, coreid.ErrDeviceIdentityNotFound
	}

	now := time.Now().UTC()
	if err := upsertPrekeysTx(ctx, tx, req.DeviceID, req.Prekeys, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return a.listActivePrekeys(ctx, req.DeviceID)
}

func (a *DeviceKeysAdapter) RevokeDeviceIdentity(ctx context.Context, userID coreid.UserID, deviceID int64) error {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `
		UPDATE device_identities
		SET key_state = ?, revoked_at = COALESCE(revoked_at, ?), updated_at = ?
		WHERE id = ? AND user_id = ? AND revoked_at IS NULL
	`, coreid.DeviceKeyStateRevoked, now, now, deviceID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return coreid.ErrDeviceIdentityNotFound
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE device_prekeys
		SET key_state = ?, revoked_at = COALESCE(revoked_at, ?), updated_at = ?
		WHERE device_identity_id = ? AND revoked_at IS NULL
	`, coreid.DeviceKeyStateRevoked, now, now, deviceID); err != nil {
		return err
	}

	return tx.Commit()
}

func (a *DeviceKeysAdapter) GetDeviceDirectory(ctx context.Context, username string) (coreid.DeviceDirectory, error) {
	var directory coreid.DeviceDirectory
	if err := a.DB.QueryRowContext(ctx, `
		SELECT id, username
		FROM users
		WHERE username = ?
	`, username).Scan(&directory.UserID, &directory.Username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return coreid.DeviceDirectory{}, store.ErrNotFound
		}
		return coreid.DeviceDirectory{}, err
	}

	rows, err := a.DB.QueryContext(ctx, `
		SELECT id, user_id, label, algorithm, identity_key, signed_prekey_id,
		       signed_prekey, signed_prekey_signature, key_state,
		       created_at, published_at, rotated_at, revoked_at
		FROM device_identities
		WHERE user_id = ? AND key_state = 'active' AND revoked_at IS NULL
		ORDER BY created_at DESC
	`, directory.UserID)
	if err != nil {
		return coreid.DeviceDirectory{}, err
	}
	defer rows.Close()

	directory.Devices = make([]coreid.DeviceDirectoryEntry, 0)
	for rows.Next() {
		device, err := scanPublicDirectoryDevice(rows)
		if err != nil {
			return coreid.DeviceDirectory{}, err
		}
		prekeys, err := a.listActivePrekeys(ctx, device.ID)
		if err != nil {
			return coreid.DeviceDirectory{}, err
		}
		device.PrekeyCount = len(prekeys)
		directory.Devices = append(directory.Devices, coreid.DeviceDirectoryEntry{
			DeviceIdentity: device,
			Prekeys:        prekeys,
		})
	}
	if err := rows.Err(); err != nil {
		return coreid.DeviceDirectory{}, err
	}
	return directory, nil
}

func (a *DeviceKeysAdapter) getDeviceIdentity(ctx context.Context, userID coreid.UserID, sessionID int64, deviceID int64) (coreid.DeviceIdentity, error) {
	row := a.DB.QueryRowContext(ctx, `
		SELECT
			di.id,
			di.user_id,
			di.label,
			di.algorithm,
			di.identity_key,
			di.signed_prekey_id,
			di.signed_prekey,
			di.signed_prekey_signature,
			di.key_state,
			di.created_at,
			di.published_at,
			di.rotated_at,
			di.revoked_at,
			COALESCE(COUNT(CASE WHEN dp.key_state = 'active' AND dp.revoked_at IS NULL THEN 1 END), 0) AS prekey_count,
			MAX(CASE WHEN ds.auth_session_id = ? THEN 1 ELSE 0 END) AS current_session
		FROM device_identities di
		LEFT JOIN device_prekeys dp ON dp.device_identity_id = di.id
		LEFT JOIN device_sessions ds ON ds.device_identity_id = di.id
		WHERE di.id = ? AND di.user_id = ?
		GROUP BY di.id
	`, sessionID, deviceID, userID)
	device, err := scanDeviceIdentity(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return coreid.DeviceIdentity{}, coreid.ErrDeviceIdentityNotFound
		}
		return coreid.DeviceIdentity{}, err
	}
	return device, nil
}

func (a *DeviceKeysAdapter) listActivePrekeys(ctx context.Context, deviceID int64) ([]coreid.DevicePrekey, error) {
	rows, err := a.DB.QueryContext(ctx, `
		SELECT id, device_identity_id, prekey_id, public_key, key_state, created_at, revoked_at
		FROM device_prekeys
		WHERE device_identity_id = ? AND key_state = 'active' AND revoked_at IS NULL
		ORDER BY prekey_id ASC
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]coreid.DevicePrekey, 0)
	for rows.Next() {
		prekey, err := scanDevicePrekey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, prekey)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func deviceBelongsToUserTx(ctx context.Context, tx *sql.Tx, deviceID int64, userID coreid.UserID) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM device_identities
		WHERE id = ? AND user_id = ? AND revoked_at IS NULL
	`, deviceID, userID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists == 1, nil
}

func upsertDeviceSessionTx(ctx context.Context, tx *sql.Tx, deviceID int64, sessionID int64, now time.Time) error {
	if sessionID <= 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO device_sessions (device_identity_id, auth_session_id, created_at, last_seen_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(device_identity_id, auth_session_id) DO UPDATE SET
			last_seen_at = excluded.last_seen_at
	`, deviceID, sessionID, now, now)
	return err
}

func upsertPrekeysTx(ctx context.Context, tx *sql.Tx, deviceID int64, prekeys []coreid.DevicePrekeyUpload, now time.Time) error {
	for _, prekey := range prekeys {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO device_prekeys (
				device_identity_id, prekey_id, public_key, key_state, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(device_identity_id, prekey_id) DO UPDATE SET
				public_key = excluded.public_key,
				key_state = excluded.key_state,
				revoked_at = NULL,
				updated_at = excluded.updated_at
		`, deviceID, prekey.PrekeyID, prekey.PublicKey, coreid.DeviceKeyStateActive, now, now); err != nil {
			return err
		}
	}
	return nil
}

type deviceIdentityScanner interface {
	Scan(dest ...any) error
}

func scanDeviceIdentity(scanner deviceIdentityScanner) (coreid.DeviceIdentity, error) {
	var device coreid.DeviceIdentity
	var keyState string
	var revokedAt sql.NullTime
	var currentSession int
	if err := scanner.Scan(
		&device.ID,
		&device.UserID,
		&device.Label,
		&device.Algorithm,
		&device.IdentityKey,
		&device.SignedPrekeyID,
		&device.SignedPrekey,
		&device.SignedPrekeySignature,
		&keyState,
		&device.CreatedAt,
		&device.PublishedAt,
		&device.RotatedAt,
		&revokedAt,
		&device.PrekeyCount,
		&currentSession,
	); err != nil {
		return coreid.DeviceIdentity{}, err
	}
	device.State = coreid.DeviceKeyState(keyState)
	device.CurrentSession = currentSession == 1
	if revokedAt.Valid {
		t := revokedAt.Time
		device.RevokedAt = &t
	}
	return device, nil
}

func scanPublicDirectoryDevice(scanner deviceIdentityScanner) (coreid.DeviceIdentity, error) {
	var device coreid.DeviceIdentity
	var keyState string
	var revokedAt sql.NullTime
	if err := scanner.Scan(
		&device.ID,
		&device.UserID,
		&device.Label,
		&device.Algorithm,
		&device.IdentityKey,
		&device.SignedPrekeyID,
		&device.SignedPrekey,
		&device.SignedPrekeySignature,
		&keyState,
		&device.CreatedAt,
		&device.PublishedAt,
		&device.RotatedAt,
		&revokedAt,
	); err != nil {
		return coreid.DeviceIdentity{}, err
	}
	device.State = coreid.DeviceKeyState(keyState)
	if revokedAt.Valid {
		t := revokedAt.Time
		device.RevokedAt = &t
	}
	return device, nil
}

func scanDevicePrekey(scanner deviceIdentityScanner) (coreid.DevicePrekey, error) {
	var prekey coreid.DevicePrekey
	var keyState string
	var revokedAt sql.NullTime
	if err := scanner.Scan(
		&prekey.ID,
		&prekey.DeviceIdentityID,
		&prekey.PrekeyID,
		&prekey.PublicKey,
		&keyState,
		&prekey.CreatedAt,
		&revokedAt,
	); err != nil {
		return coreid.DevicePrekey{}, err
	}
	prekey.State = coreid.DeviceKeyState(keyState)
	if revokedAt.Valid {
		t := revokedAt.Time
		prekey.RevokedAt = &t
	}
	return prekey, nil
}
