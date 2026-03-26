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
	res, err := s.DB.Exec(`INSERT INTO users (username, password_hash) VALUES (?, ?)`, username, "test-hash")
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return int(id)
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

func TestAdapter_SaveDirectMessage_PersistsEnvelopeMetadata(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID:        aliceID,
		ToUserID:          bobID,
		Body:              "",
		ContentKind:       "text",
		Ciphertext:        "opaque-envelope",
		EnvelopeVersion:   "x3dh-dr-v1",
		SenderDeviceID:    101,
		RecipientDeviceID: 202,
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage error: %v", err)
	}

	inbox, err := a.ListInbox(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListInbox error: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("expected 1 inbox message, got %d", len(inbox))
	}
	if inbox[0].ID != saved.ID {
		t.Fatalf("message id = %d, want %d", inbox[0].ID, saved.ID)
	}
	if inbox[0].Ciphertext != "opaque-envelope" || inbox[0].EnvelopeVersion != "x3dh-dr-v1" {
		t.Fatalf("unexpected encrypted payload: %+v", inbox[0])
	}
	if inbox[0].SenderDeviceID != 101 || inbox[0].RecipientDeviceID != 202 {
		t.Fatalf("unexpected device metadata: %+v", inbox[0])
	}
	if inbox[0].ContentKind != "text" {
		t.Fatalf("content kind = %q, want text", inbox[0].ContentKind)
	}
}

func TestAdapter_SaveDirectMessage_StripsPlaintextWhenEncryptedFlagDisabled(t *testing.T) {
	t.Setenv("MESSAGING_STORE_PLAINTEXT_WHEN_ENCRYPTED", "false")
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID:        aliceID,
		ToUserID:          bobID,
		Body:              "hello",
		ContentKind:       "text",
		Ciphertext:        "opaque-envelope",
		EnvelopeVersion:   "x3dh-dr-v1",
		SenderDeviceID:    101,
		RecipientDeviceID: 202,
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage error: %v", err)
	}
	if saved.Body != "" {
		t.Fatalf("saved body = %q, want empty", saved.Body)
	}

	inbox, err := a.ListInbox(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListInbox error: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("expected 1 inbox message, got %d", len(inbox))
	}
	if inbox[0].Body != "" {
		t.Fatalf("inbox body = %q, want empty", inbox[0].Body)
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

func TestAdapter_ListUnreadInbox_ExcludesPaymentUpdateControlMessages(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	if _, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID:  aliceID,
		ToUserID:    bobID,
		Body:        `__microapp_v1__:{"kind":"payment_request_update","requestId":"payreq_1","status":"paid"}`,
		ContentKind: "payment_request_update",
	}); err != nil {
		t.Fatalf("SaveDirectMessage payment update: %v", err)
	}
	visibleMsg, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "visible-message",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage visible msg: %v", err)
	}

	inbox, err := a.ListUnreadInbox(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListUnreadInbox error: %v", err)
	}
	if len(inbox) != 1 {
		t.Fatalf("expected 1 unread visible message, got %d", len(inbox))
	}
	if inbox[0].ID != visibleMsg.ID || inbox[0].Body != "visible-message" {
		t.Fatalf("unexpected unread inbox contents: %+v", inbox)
	}
}

func TestAdapter_RecordClientMessageCorrelation_UpsertsBySenderAndClientMessageID(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}
	msg1, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "first",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage first: %v", err)
	}
	msg2, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "second",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage second: %v", err)
	}

	if err := a.RecordClientMessageCorrelation(context.Background(), coremsg.ClientMessageCorrelation{
		SenderUserID:    aliceID,
		RecipientUserID: bobID,
		ClientMessageID: 12345,
		StoredMessageID: msg1.ID,
		Delivered:       false,
	}); err != nil {
		t.Fatalf("RecordClientMessageCorrelation insert: %v", err)
	}

	if err := a.RecordClientMessageCorrelation(context.Background(), coremsg.ClientMessageCorrelation{
		SenderUserID:    aliceID,
		RecipientUserID: bobID,
		ClientMessageID: 12345,
		StoredMessageID: msg2.ID,
		Delivered:       true,
	}); err != nil {
		t.Fatalf("RecordClientMessageCorrelation upsert: %v", err)
	}

	var senderID, recipientID int
	var clientMsgID, storedMsgID int64
	var deliveredInt int
	if err := s.DB.QueryRow(`
		SELECT sender_user_id, recipient_user_id, client_message_id, stored_message_id, delivered
		FROM message_client_correlations
		WHERE sender_user_id = ? AND client_message_id = ?
	`, aliceID, 12345).Scan(&senderID, &recipientID, &clientMsgID, &storedMsgID, &deliveredInt); err != nil {
		t.Fatalf("query correlation row: %v", err)
	}
	if senderID != aliceID || recipientID != bobID {
		t.Fatalf("unexpected sender/recipient ids: %d/%d", senderID, recipientID)
	}
	if clientMsgID != 12345 || storedMsgID != msg2.ID {
		t.Fatalf("unexpected ids client=%d stored=%d want stored=%d", clientMsgID, storedMsgID, msg2.ID)
	}
	if deliveredInt != 1 {
		t.Fatalf("delivered = %d, want 1", deliveredInt)
	}
}

func TestAdapter_ListOutbox_ReturnsSentMessagesDescendingByID(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	charlieID := seedUser(t, s, "charlie")
	a := &Adapter{DB: s.DB}

	saved1, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: bobID,
		ToUserID:   aliceID,
		Body:       "to-alice-1",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage 1: %v", err)
	}
	saved2, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: charlieID,
		ToUserID:   aliceID,
		Body:       "not-bob",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage 2: %v", err)
	}
	_ = saved2
	saved3, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: bobID,
		ToUserID:   charlieID,
		Body:       "to-charlie",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage 3: %v", err)
	}
	if err := a.RecordClientMessageCorrelation(context.Background(), coremsg.ClientMessageCorrelation{
		SenderUserID:    bobID,
		RecipientUserID: charlieID,
		ClientMessageID: 9876,
		StoredMessageID: saved3.ID,
		Delivered:       true,
	}); err != nil {
		t.Fatalf("RecordClientMessageCorrelation: %v", err)
	}

	outbox, err := a.ListOutbox(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListOutbox error: %v", err)
	}
	if len(outbox) != 2 {
		t.Fatalf("expected 2 bob outbox messages, got %d", len(outbox))
	}
	if outbox[0].ID != saved3.ID || outbox[1].ID != saved1.ID {
		t.Fatalf("unexpected outbox order/contents: %+v", outbox)
	}
	if outbox[0].ClientMessageID != 9876 {
		t.Fatalf("outbox[0].ClientMessageID = %d, want 9876", outbox[0].ClientMessageID)
	}
	for _, msg := range outbox {
		if msg.FromUserID != bobID {
			t.Fatalf("unexpected outbox sender: %+v", msg)
		}
	}
}

func TestAdapter_ListOutbox_ExposesDurableDeliveryFailureState(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: bobID,
		ToUserID:   aliceID,
		Body:       "offline-send",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage: %v", err)
	}

	if err := a.RecordClientMessageCorrelation(context.Background(), coremsg.ClientMessageCorrelation{
		SenderUserID:    bobID,
		RecipientUserID: aliceID,
		ClientMessageID: 555,
		StoredMessageID: saved.ID,
		Delivered:       false,
	}); err != nil {
		t.Fatalf("RecordClientMessageCorrelation: %v", err)
	}

	outbox, err := a.ListOutbox(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListOutbox error: %v", err)
	}
	if len(outbox) != 1 {
		t.Fatalf("expected 1 outbox message, got %d", len(outbox))
	}
	if !outbox[0].DeliveryFailed {
		t.Fatalf("DeliveryFailed = %v, want true", outbox[0].DeliveryFailed)
	}

	if err := a.MarkDelivered(context.Background(), saved.ID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkDelivered: %v", err)
	}

	outbox, err = a.ListOutbox(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListOutbox after delivery error: %v", err)
	}
	if outbox[0].DeliveryFailed {
		t.Fatalf("DeliveryFailed = %v, want false after delivery", outbox[0].DeliveryFailed)
	}
}

func TestAdapter_ListOutboxBeforeAndAfter_PaginatesByMessageID(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	var ids []int64
	for _, body := range []string{"m1", "m2", "m3"} {
		saved, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
			FromUserID: bobID,
			ToUserID:   aliceID,
			Body:       body,
		})
		if err != nil {
			t.Fatalf("SaveDirectMessage error: %v", err)
		}
		ids = append(ids, saved.ID)
	}

	beforePage, err := a.ListOutboxBefore(context.Background(), bobID, ids[2], 10)
	if err != nil {
		t.Fatalf("ListOutboxBefore error: %v", err)
	}
	if len(beforePage) != 2 {
		t.Fatalf("expected 2 messages before latest, got %d", len(beforePage))
	}
	if beforePage[0].ID != ids[1] || beforePage[1].ID != ids[0] {
		t.Fatalf("unexpected outbox before-page: %+v", beforePage)
	}

	afterPage, err := a.ListOutboxAfter(context.Background(), bobID, ids[0], 10)
	if err != nil {
		t.Fatalf("ListOutboxAfter error: %v", err)
	}
	if len(afterPage) != 2 {
		t.Fatalf("expected 2 messages after first, got %d", len(afterPage))
	}
	if afterPage[0].ID != ids[1] || afterPage[1].ID != ids[2] {
		t.Fatalf("unexpected outbox after-page: %+v", afterPage)
	}
}

func TestAdapter_ListThreadSummaries_ReturnsLatestMessageAndUnreadCounts(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	charlieID := seedUser(t, s, "charlie")
	if _, err := s.DB.Exec(`UPDATE users SET display_name = ?, avatar_url = ? WHERE id = ?`, "Alice", "https://example.com/alice.png", aliceID); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{DB: s.DB}

	msg1, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "alice-1",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage alice-1: %v", err)
	}
	msg2, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: bobID,
		ToUserID:   aliceID,
		Body:       "bob-2",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage bob-2: %v", err)
	}
	if err := a.MarkRead(context.Background(), msg2.ID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkRead bob-2: %v", err)
	}
	msg3, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: charlieID,
		ToUserID:   bobID,
		Body:       "charlie-3",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage charlie-3: %v", err)
	}

	summaries, err := a.ListThreadSummaries(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListThreadSummaries error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	if summaries[0].CounterpartyUsername != "charlie" || summaries[0].LastMessageID != msg3.ID || summaries[0].UnreadCount != 1 {
		t.Fatalf("unexpected first summary: %+v", summaries[0])
	}
	if summaries[1].CounterpartyUsername != "alice" || summaries[1].LastMessageID != msg2.ID {
		t.Fatalf("unexpected second summary: %+v", summaries[1])
	}
	if summaries[1].UnreadCount != 1 {
		t.Fatalf("alice unread_count = %d, want 1", summaries[1].UnreadCount)
	}
	if summaries[1].CounterpartyDisplayName != "Alice" || summaries[1].CounterpartyAvatarURL != "https://example.com/alice.png" {
		t.Fatalf("missing counterparty metadata: %+v", summaries[1])
	}
	if summaries[1].LastReadAt == nil {
		t.Fatalf("expected last read_at on alice summary: %+v", summaries[1])
	}
	if msg1.ID == summaries[1].LastMessageID {
		t.Fatalf("expected latest alice thread message to be msg2, got %+v", summaries[1])
	}
}

func TestAdapter_ListThreadSummaries_IgnorePaymentUpdateControlMessages(t *testing.T) {
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	visibleMsg, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID: aliceID,
		ToUserID:   bobID,
		Body:       "request sent",
	})
	if err != nil {
		t.Fatalf("SaveDirectMessage visible msg: %v", err)
	}
	if err := a.MarkRead(context.Background(), visibleMsg.ID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkRead visible msg: %v", err)
	}

	if _, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID:  aliceID,
		ToUserID:    bobID,
		Body:        `__microapp_v1__:{"kind":"payment_request_update","requestId":"payreq_1","status":"paid"}`,
		ContentKind: "payment_request_update",
	}); err != nil {
		t.Fatalf("SaveDirectMessage payment update: %v", err)
	}

	summaries, err := a.ListThreadSummaries(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListThreadSummaries error: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].LastMessageID != visibleMsg.ID {
		t.Fatalf("LastMessageID = %d, want %d", summaries[0].LastMessageID, visibleMsg.ID)
	}
	if summaries[0].LastMessageBody != "request sent" {
		t.Fatalf("LastMessageBody = %q, want request sent", summaries[0].LastMessageBody)
	}
	if summaries[0].UnreadCount != 0 {
		t.Fatalf("UnreadCount = %d, want 0", summaries[0].UnreadCount)
	}
}

func TestAdapter_ListThreadSummaries_EncryptedMessageUsesPlaceholderPreview(t *testing.T) {
	t.Setenv("MESSAGING_STORE_PLAINTEXT_WHEN_ENCRYPTED", "false")
	s := newMessagingStore(t)
	aliceID := seedUser(t, s, "alice")
	bobID := seedUser(t, s, "bob")
	a := &Adapter{DB: s.DB}

	if _, err := a.SaveDirectMessage(context.Background(), coremsg.StoredMessage{
		FromUserID:        aliceID,
		ToUserID:          bobID,
		Body:              "hidden",
		ContentKind:       "text",
		Ciphertext:        "opaque-envelope",
		EnvelopeVersion:   "x3dh-dr-v1",
		SenderDeviceID:    7,
		RecipientDeviceID: 9,
	}); err != nil {
		t.Fatalf("SaveDirectMessage encrypted msg: %v", err)
	}

	summaries, err := a.ListThreadSummaries(context.Background(), bobID, 10)
	if err != nil {
		t.Fatalf("ListThreadSummaries error: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].LastMessageBody != "Encrypted message" {
		t.Fatalf("LastMessageBody = %q, want Encrypted message", summaries[0].LastMessageBody)
	}
	if summaries[0].LastMessageContentKind != "text" {
		t.Fatalf("LastMessageContentKind = %q, want text", summaries[0].LastMessageContentKind)
	}
}
