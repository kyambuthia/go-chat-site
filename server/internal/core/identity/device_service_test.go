package identity

import (
	"context"
	"errors"
	"testing"
)

type fakeDeviceIdentityRepo struct {
	device     DeviceIdentity
	devices    []DeviceIdentity
	prekeys    []DevicePrekey
	directory  DeviceDirectory
	err        error
	lastUserID UserID
	lastSessID int64
	lastReg    RegisterDeviceIdentityRequest
	lastRotate RotateDeviceIdentityRequest
	lastPub    PublishPrekeysRequest
	lastRevoke int64
	lastLookup string
}

func (f *fakeDeviceIdentityRepo) CreateDeviceIdentity(ctx context.Context, userID UserID, sessionID int64, req RegisterDeviceIdentityRequest) (DeviceIdentity, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastSessID = sessionID
	f.lastReg = req
	return f.device, f.err
}

func (f *fakeDeviceIdentityRepo) ListDeviceIdentities(ctx context.Context, userID UserID, sessionID int64) ([]DeviceIdentity, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastSessID = sessionID
	return f.devices, f.err
}

func (f *fakeDeviceIdentityRepo) RotateDeviceIdentity(ctx context.Context, userID UserID, sessionID int64, req RotateDeviceIdentityRequest) (DeviceIdentity, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastSessID = sessionID
	f.lastRotate = req
	return f.device, f.err
}

func (f *fakeDeviceIdentityRepo) PublishPrekeys(ctx context.Context, userID UserID, req PublishPrekeysRequest) ([]DevicePrekey, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastPub = req
	return f.prekeys, f.err
}

func (f *fakeDeviceIdentityRepo) RevokeDeviceIdentity(ctx context.Context, userID UserID, deviceID int64) error {
	_ = ctx
	f.lastUserID = userID
	f.lastRevoke = deviceID
	return f.err
}

func (f *fakeDeviceIdentityRepo) GetDeviceDirectory(ctx context.Context, username string) (DeviceDirectory, error) {
	_ = ctx
	f.lastLookup = username
	return f.directory, f.err
}

func TestDeviceIdentityService_RegisterDelegatesAndNormalizes(t *testing.T) {
	repo := &fakeDeviceIdentityRepo{device: DeviceIdentity{ID: 3, Label: "This device"}}
	svc := NewDeviceIdentityService(repo)

	got, err := svc.RegisterDeviceIdentity(context.Background(), 7, 9, RegisterDeviceIdentityRequest{
		IdentityKey:           "identity-key",
		SignedPrekeyID:        1,
		SignedPrekey:          "signed-prekey",
		SignedPrekeySignature: "signature",
		Prekeys: []DevicePrekeyUpload{
			{PrekeyID: 1, PublicKey: "prekey-1"},
		},
	})
	if err != nil {
		t.Fatalf("RegisterDeviceIdentity error: %v", err)
	}
	if got.ID != 3 {
		t.Fatalf("unexpected device: %+v", got)
	}
	if repo.lastReg.Label != "This device" {
		t.Fatalf("label = %q, want This device", repo.lastReg.Label)
	}
	if repo.lastReg.Algorithm != MessagingKeyAlgorithmX3DHV1 {
		t.Fatalf("algorithm = %q, want %q", repo.lastReg.Algorithm, MessagingKeyAlgorithmX3DHV1)
	}
}

func TestDeviceIdentityService_RegisterRejectsInvalidPayload(t *testing.T) {
	svc := NewDeviceIdentityService(&fakeDeviceIdentityRepo{})

	if _, err := svc.RegisterDeviceIdentity(context.Background(), 7, 9, RegisterDeviceIdentityRequest{}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDeviceIdentityService_PublishPrekeysRejectsDuplicates(t *testing.T) {
	svc := NewDeviceIdentityService(&fakeDeviceIdentityRepo{})

	_, err := svc.PublishPrekeys(context.Background(), 7, PublishPrekeysRequest{
		DeviceID: 9,
		Prekeys: []DevicePrekeyUpload{
			{PrekeyID: 1, PublicKey: "a"},
			{PrekeyID: 1, PublicKey: "b"},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate prekey validation error")
	}
}

func TestDeviceIdentityService_RotateDelegates(t *testing.T) {
	repo := &fakeDeviceIdentityRepo{device: DeviceIdentity{ID: 4}}
	svc := NewDeviceIdentityService(repo)

	_, err := svc.RotateDeviceIdentity(context.Background(), 8, 11, RotateDeviceIdentityRequest{
		DeviceID:              4,
		SignedPrekeyID:        2,
		SignedPrekey:          "rotated-prekey",
		SignedPrekeySignature: "rotated-signature",
	})
	if err != nil {
		t.Fatalf("RotateDeviceIdentity error: %v", err)
	}
	if repo.lastRotate.DeviceID != 4 {
		t.Fatalf("deviceID = %d, want 4", repo.lastRotate.DeviceID)
	}
}

func TestDeviceIdentityService_DirectoryDelegates(t *testing.T) {
	repo := &fakeDeviceIdentityRepo{directory: DeviceDirectory{Username: "alice"}}
	svc := NewDeviceIdentityService(repo)

	got, err := svc.GetDeviceDirectory(context.Background(), " alice ")
	if err != nil {
		t.Fatalf("GetDeviceDirectory error: %v", err)
	}
	if repo.lastLookup != "alice" {
		t.Fatalf("lookup username = %q, want alice", repo.lastLookup)
	}
	if got.Username != "alice" {
		t.Fatalf("unexpected directory: %+v", got)
	}
}

func TestDeviceIdentityService_PropagatesRepositoryErrors(t *testing.T) {
	wantErr := errors.New("db down")
	repo := &fakeDeviceIdentityRepo{err: wantErr}
	svc := NewDeviceIdentityService(repo)

	if err := svc.RevokeDeviceIdentity(context.Background(), 7, 3); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
