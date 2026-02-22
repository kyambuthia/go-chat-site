package identity

import "context"

type ProfileRepository interface {
	GetProfile(ctx context.Context, userID UserID) (Profile, error)
}

type ProfileService interface {
	GetProfile(ctx context.Context, userID UserID) (Profile, error)
}

type profileService struct {
	repo ProfileRepository
}

func NewProfileService(repo ProfileRepository) ProfileService {
	return &profileService{repo: repo}
}

func (s *profileService) GetProfile(ctx context.Context, userID UserID) (Profile, error) {
	return s.repo.GetProfile(ctx, userID)
}
