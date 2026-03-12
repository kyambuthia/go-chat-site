package messaging

import (
	"context"
	"errors"
	"testing"
)

type fakeThreadSummaryRepo struct {
	summaries  []ThreadSummary
	err        error
	lastUserID int
	lastLimit  int
}

func (f *fakeThreadSummaryRepo) ListThreadSummaries(ctx context.Context, userID int, limit int) ([]ThreadSummary, error) {
	_ = ctx
	f.lastUserID = userID
	f.lastLimit = limit
	return f.summaries, f.err
}

func TestThreadSummaryService_ListThreadSummaries_UsesDefaultLimitAndDelegates(t *testing.T) {
	repo := &fakeThreadSummaryRepo{summaries: []ThreadSummary{{CounterpartyUsername: "alice"}}}
	svc := NewThreadSummaryService(repo)

	got, err := svc.ListThreadSummaries(context.Background(), 7, 0)
	if err != nil {
		t.Fatalf("ListThreadSummaries error: %v", err)
	}
	if repo.lastUserID != 7 || repo.lastLimit != 100 {
		t.Fatalf("unexpected repo call user=%d limit=%d", repo.lastUserID, repo.lastLimit)
	}
	if len(got) != 1 || got[0].CounterpartyUsername != "alice" {
		t.Fatalf("unexpected summaries: %+v", got)
	}
}

func TestThreadSummaryService_ListThreadSummaries_PropagatesErrors(t *testing.T) {
	repo := &fakeThreadSummaryRepo{err: errors.New("db down")}
	svc := NewThreadSummaryService(repo)

	if _, err := svc.ListThreadSummaries(context.Background(), 1, 10); err == nil {
		t.Fatal("expected error")
	}
}
