package sqliteidentity

import (
	"context"

	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type Adapter struct {
	Store store.MeStore
}

var _ coreid.ProfileRepository = (*Adapter)(nil)

func (a *Adapter) GetProfile(ctx context.Context, userID coreid.UserID) (coreid.Profile, error) {
	_ = ctx
	user, err := a.Store.GetUserByID(int(userID))
	if err != nil {
		return coreid.Profile{}, err
	}
	return coreid.Profile{
		UserID:      coreid.UserID(user.ID),
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
	}, nil
}
