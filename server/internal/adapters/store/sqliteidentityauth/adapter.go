package sqliteidentityauth

import (
	"context"

	coreid "github.com/kyambuthia/go-chat-site/server/internal/core/identity"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

type Dependencies interface {
	store.AuthStore
	store.LoginStore
}

type Adapter struct {
	Store Dependencies
}

var _ coreid.AuthRepository = (*Adapter)(nil)

func (a *Adapter) CreateUser(ctx context.Context, cred coreid.PasswordCredential) (coreid.Principal, error) {
	_ = ctx
	id, err := a.Store.CreateUser(cred.Username, cred.Password)
	if err != nil {
		return coreid.Principal{}, err
	}
	return coreid.Principal{ID: coreid.UserID(id), Username: cred.Username}, nil
}

func (a *Adapter) GetPasswordLoginRecord(ctx context.Context, username string) (coreid.PasswordLoginRecord, error) {
	_ = ctx
	user, err := a.Store.GetUserByUsername(username)
	if err != nil {
		return coreid.PasswordLoginRecord{}, err
	}
	return coreid.PasswordLoginRecord{
		Principal:    coreid.Principal{ID: coreid.UserID(user.ID), Username: user.Username},
		PasswordHash: user.PasswordHash,
	}, nil
}
