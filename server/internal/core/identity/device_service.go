package identity

import (
	"context"
	"errors"
	"strings"
)

type DeviceIdentityRepository interface {
	CreateDeviceIdentity(ctx context.Context, userID UserID, sessionID int64, req RegisterDeviceIdentityRequest) (DeviceIdentity, error)
	ListDeviceIdentities(ctx context.Context, userID UserID, sessionID int64) ([]DeviceIdentity, error)
	RotateDeviceIdentity(ctx context.Context, userID UserID, sessionID int64, req RotateDeviceIdentityRequest) (DeviceIdentity, error)
	PublishPrekeys(ctx context.Context, userID UserID, req PublishPrekeysRequest) ([]DevicePrekey, error)
	RevokeDeviceIdentity(ctx context.Context, userID UserID, deviceID int64) error
	GetDeviceDirectory(ctx context.Context, username string) (DeviceDirectory, error)
}

type DeviceIdentityService interface {
	RegisterDeviceIdentity(ctx context.Context, userID UserID, sessionID int64, req RegisterDeviceIdentityRequest) (DeviceIdentity, error)
	ListDeviceIdentities(ctx context.Context, userID UserID, sessionID int64) ([]DeviceIdentity, error)
	RotateDeviceIdentity(ctx context.Context, userID UserID, sessionID int64, req RotateDeviceIdentityRequest) (DeviceIdentity, error)
	PublishPrekeys(ctx context.Context, userID UserID, req PublishPrekeysRequest) ([]DevicePrekey, error)
	RevokeDeviceIdentity(ctx context.Context, userID UserID, deviceID int64) error
	GetDeviceDirectory(ctx context.Context, username string) (DeviceDirectory, error)
}

type deviceIdentityService struct {
	repo DeviceIdentityRepository
}

func NewDeviceIdentityService(repo DeviceIdentityRepository) DeviceIdentityService {
	return &deviceIdentityService{repo: repo}
}

func (s *deviceIdentityService) RegisterDeviceIdentity(ctx context.Context, userID UserID, sessionID int64, req RegisterDeviceIdentityRequest) (DeviceIdentity, error) {
	if s.repo == nil {
		return DeviceIdentity{}, errors.New("device identity repository unavailable")
	}
	req.Label = normalizeDeviceIdentityLabel(req.Label)
	req.Algorithm = normalizeDeviceIdentityAlgorithm(req.Algorithm)
	if err := validateDeviceRegistration(req); err != nil {
		return DeviceIdentity{}, err
	}
	return s.repo.CreateDeviceIdentity(ctx, userID, sessionID, req)
}

func (s *deviceIdentityService) ListDeviceIdentities(ctx context.Context, userID UserID, sessionID int64) ([]DeviceIdentity, error) {
	if s.repo == nil {
		return nil, errors.New("device identity repository unavailable")
	}
	return s.repo.ListDeviceIdentities(ctx, userID, sessionID)
}

func (s *deviceIdentityService) RotateDeviceIdentity(ctx context.Context, userID UserID, sessionID int64, req RotateDeviceIdentityRequest) (DeviceIdentity, error) {
	if s.repo == nil {
		return DeviceIdentity{}, errors.New("device identity repository unavailable")
	}
	if req.DeviceID <= 0 {
		return DeviceIdentity{}, errors.New("device_id is required")
	}
	if strings.TrimSpace(req.SignedPrekey) == "" {
		return DeviceIdentity{}, errors.New("signed_prekey is required")
	}
	if strings.TrimSpace(req.SignedPrekeySignature) == "" {
		return DeviceIdentity{}, errors.New("signed_prekey_signature is required")
	}
	if req.SignedPrekeyID <= 0 {
		return DeviceIdentity{}, errors.New("signed_prekey_id is required")
	}
	if err := validateDevicePrekeys(req.Prekeys, false); err != nil {
		return DeviceIdentity{}, err
	}
	return s.repo.RotateDeviceIdentity(ctx, userID, sessionID, req)
}

func (s *deviceIdentityService) PublishPrekeys(ctx context.Context, userID UserID, req PublishPrekeysRequest) ([]DevicePrekey, error) {
	if s.repo == nil {
		return nil, errors.New("device identity repository unavailable")
	}
	if req.DeviceID <= 0 {
		return nil, errors.New("device_id is required")
	}
	if err := validateDevicePrekeys(req.Prekeys, true); err != nil {
		return nil, err
	}
	return s.repo.PublishPrekeys(ctx, userID, req)
}

func (s *deviceIdentityService) RevokeDeviceIdentity(ctx context.Context, userID UserID, deviceID int64) error {
	if s.repo == nil {
		return errors.New("device identity repository unavailable")
	}
	if deviceID <= 0 {
		return errors.New("device_id is required")
	}
	return s.repo.RevokeDeviceIdentity(ctx, userID, deviceID)
}

func (s *deviceIdentityService) GetDeviceDirectory(ctx context.Context, username string) (DeviceDirectory, error) {
	if s.repo == nil {
		return DeviceDirectory{}, errors.New("device identity repository unavailable")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return DeviceDirectory{}, errors.New("username is required")
	}
	return s.repo.GetDeviceDirectory(ctx, username)
}

func normalizeDeviceIdentityLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "This device"
	}
	return label
}

func normalizeDeviceIdentityAlgorithm(algorithm string) string {
	algorithm = strings.TrimSpace(algorithm)
	if algorithm == "" {
		return MessagingKeyAlgorithmX3DHV1
	}
	return algorithm
}

func validateDeviceRegistration(req RegisterDeviceIdentityRequest) error {
	if strings.TrimSpace(req.IdentityKey) == "" {
		return errors.New("identity_key is required")
	}
	if strings.TrimSpace(req.SignedPrekey) == "" {
		return errors.New("signed_prekey is required")
	}
	if strings.TrimSpace(req.SignedPrekeySignature) == "" {
		return errors.New("signed_prekey_signature is required")
	}
	if req.SignedPrekeyID <= 0 {
		return errors.New("signed_prekey_id is required")
	}
	return validateDevicePrekeys(req.Prekeys, true)
}

func validateDevicePrekeys(prekeys []DevicePrekeyUpload, required bool) error {
	if required && len(prekeys) == 0 {
		return errors.New("at least one prekey is required")
	}
	seen := make(map[int64]struct{}, len(prekeys))
	for _, prekey := range prekeys {
		if prekey.PrekeyID <= 0 {
			return errors.New("prekey_id is required")
		}
		if strings.TrimSpace(prekey.PublicKey) == "" {
			return errors.New("prekey public_key is required")
		}
		if _, ok := seen[prekey.PrekeyID]; ok {
			return errors.New("duplicate prekey_id")
		}
		seen[prekey.PrekeyID] = struct{}{}
	}
	return nil
}
