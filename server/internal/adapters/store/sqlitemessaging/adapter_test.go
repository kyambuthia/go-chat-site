package sqlitemessaging

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	coremsg "github.com/kyambuthia/go-chat-site/server/internal/core/messaging"
	"github.com/kyambuthia/go-chat-site/server/internal/migrate"
	"github.com/kyambuthia/go-chat-site/server/internal/store"
)

func newMessagingStore(t *testing.T) *store.SqliteStore {
	t.Helper()
	s, err := store.NewSqliteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })
	if err := migrate.RunMigrations(s.DB, filepath.Join("..", "..", "..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	return s
}

func seedUser(t *testing.T, s *store.SqliteStore, username string) int {
	t.Helper()
	id, err := s.CreateUser(username, "password123")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestAdapter_SaveDirectMessage_AndListInbox(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "hello",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage error: %v", err)
	}
	if saved.ID == 0 {
		t.Fatal("expected non-zero message ID")
	}

	inbox, err := a.ListInbox(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListInbox error: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("expected 1 inbox message, got %d", len(inbox))
	}
	if inbox[0].ID != saved.ID || inbox[0].FromUserID != aliceID || inbox[0].ToUserID != bobID || inbox[0].Body != "hello" {
		t.Fatalf("unexpected inbox message: %+v", inbox[0])
	}
}

func TestAdapter_MarkDeliveredAndRead_UpsertsMessageDeliveries(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "hello",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage error: %v", err)
	}

	deliveredAt := time.Now().UTC().Truncate(time.Second)
	if err := a.MarkDelivered(context.Background(), saved.ID, deliveredAt); err != nil {
		t.Fatalf("MarkDelivered error: %v", err)
	}

	readAt := deliveredAt.Add(2 * time.Minute)
	if err := a.MarkRead(context.Background(), saved.ID, readAt); err != nil {
		t.Fatalf("MarkRead error: %v", err)
	}

	inbox, err := a.ListInbox(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListInbox error: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("expected 1 inbox message, got %d", len(inbox))
	}
	if inbox[0].DeliveredAt == nil || inbox[0].ReadAt == nil {
		t.Fatalf("expected delivered/read timestamps, got %+v", inbox[0])
	}
	if inbox[0].DeliveredAt.IsZero() || inbox[0].ReadAt.IsZero() {
		t.Fatalf("expected non-zero delivered/read timestamps, got %+v", inbox[0])
	}
}

func TestAdapter_MarkRead_WithoutPriorDeliverySetsDeliveredToo(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "hello",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage error: %v", err)
	}

	if err := a.MarkRead(context.Background(), saved.ID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkRead error: %v", err)
	}

	inbox, err := a.ListInbox(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListInbox error: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("expected 1 inbox message, got %d", len(inbox))
	}
	if inbox[0].DeliveredAt == nil || inbox[0].ReadAt == nil {
		t.Fatalf("expected delivered/read timestamps after read, got %+v", inbox[0])
	}
}

func TestAdapter_RecipientScopedReceipts_RejectWrongRecipient(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	charlieID := seedUser(t, s, "charlie")
	a := &Adapter{DB: s.DB}

	saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "hello",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage error: %v", err)
	}

	if err := a.MarkDeliveredForRecipient(context.Background(), charlieID, saved.ID, time.Now().UTC()); !errors.Is(err, coremsg.ErrMessageNotFound) {
		t.Fatalf("expected ErrMessageNotFound for wrong recipient delivered, got %v", err)
	}
	if err := a.MarkReadForRecipient(context.Background(), charlieID, saved.ID, time.Now().UTC()); !errors.Is(err, coremsg.ErrMessageNotFound) {
		t.Fatalf("expected ErrMessageNotFound for wrong recipient read, got %v", err)
	}
}

func TestAdapter_ListInboxBefore_PaginatesByDescendingMessageID(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	var ids []int64
	for _, body := range []string{"m1", "m2", "m3"} {
		saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
			FromUserID: aliceID,
			ToUserID:   bobID,
			Body:       body,
		})
		if err != nil {
			t.Fatalf("SaveDirectMessage error: %v", err)
		}
		ids = append(ids, saved.ID)
	}

	page, err := a.ListInboxBefore(context.Background(), bobID, ids[2], 10)
	if err != nil {
		t.Fatalf("ListInboxBefore error: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected 2 messages before latest, got %d", len(page))
	}
	if page[0].ID != ids[1] || page[1].ID != ids[0] {
		t.Fatalf("unexpected page order/contents: %+v", page)
	}
}

func TestAdapter_ListInboxAfter_ReturnsNewerMessagesAscendingByID(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	var ids []int64
	for _, body := range []string{"m1", "m2", "m3"} {
		saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
			FromUserID: aliceID,
			ToUserID:   bobID,
			Body:       body,
		})
		if err != nil {
			t.Fatalf("SaveDirectMessage error: %v", err)
		}
		ids = append(ids, saved.ID)
	}

	page, err := a.ListInboxAfter(context.Background(), bobID, ids[0], 10)
	if err != nil {
		t.Fatalf("ListInboxAfter error: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected 2 messages after first, got %d", len(page))
	}
	if page[0].ID != ids[1] || page[1].ID != ids[2] {
		t.Fatalf("unexpected ascending page order/contents: %+v", page)
	}
}

func TestAdapter_ListInboxWithUser_FiltersByCounterparty(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	charlieID := seedUser(t, s, "charlie")
	a := &Adapter{DB: s.DB}

	var aliceMsgID int64
	for _, tc := range []struct {
		from int
		body string
	}{
		{from: aliceID, body: "from-alice-1"},
		{from: charlieID, body: "from-charlie"},
		{from: aliceID, body: "from-alice-2"},
	} {
		saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
			FromUserID: tc.from,
			ToUserID:   bobID,
			Body:       tc.body,
		})
		if err != nil {
			t.Fatalf("SaveDirectMessage error: %v", err)
		}
		if tc.body == "from-alice-2" {
			aliceMsgID = saved.ID
		}
	}

	inbox, err := a.ListInboxWithUser(context.Background(), bobID, aliceID, 10)
	if err != nil {
		t.Fatalf("ListInboxWithUser error: %v", err)
	}
	if len(inbox) != 2 {
		t.Fatalf("expected 2 alice messages, got %d", len(inbox))
	}
	for _, msg := range inbox {
		if msg.FromUserID != aliceID {
			t.Fatalf("unexpected sender in filtered inbox: %+v", msg)
		}
	}

	page, err := a.ListInboxBeforeWithUser(context.Background(), bobID, aliceID, aliceMsgID, 10)
	if err != nil {
		t.Fatalf("ListInboxBeforeWithUser error: %v", err)
	}
	if len(page) != 1 || page[0].Body != "from-alice-1" {
		t.Fatalf("unexpected filtered before-page: %+v", page)
	}
}

func TestAdapter_ListUnreadInbox_FiltersReadMessages(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	readMsg, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "read-message",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage read msg: %v", err)
	}
	unreadMsg, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "unread-message",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage unread msg: %v", err)
	}

	if err := a.MarkRead(context.Background(), readMsg.ID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkRead error: %v", err)
	}

	inbox, err := a.ListUnreadInbox(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListUnreadInbox error: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("expected 1 unread message, got %d", len(inbox))
	}
	if inbox[0].ID != unreadMsg.ID || inbox[0].Body != "unread-message" {
		t.Fatalf("unexpected unread inbox contents: %+v", inbox)
	}
	if inbox[0].ReadAt != nil {
		t.Fatalf("expected unread message to have nil read_at, got %+v", inbox[0])
	}
}
