package messaging

import (
	"context"
	"testing"
)

type fakeTransport struct {
	ok      bool
	toUser  int
	lastMsg Message
}

func (f *fakeTransport) SendDirect(toUserID int, msg Message) bool {
	f.toUser = toUserID
	f.lastMsg = msg
	return f.ok
}

func TestRelayService_SendDirect_ReturnsDeliveredReceipt(t *testing.T) {
	tp := &fakeTransport{ok: true}
	svc := NewRelayService(tp)

	receipt, err := svc.SendDirect(context.Background(), DirectSendRequest{
		FromUserID: 1,
		From:       "alice",
		ToUserID:   2,
		Body:       "hello",
		MessageID:  44,
	})
	if err != nil {
		t.Fatalf("SendDirect returned error: %v", err)
	}
	if !receipt.Delivered || receipt.MessageID != 44 {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	if tp.toUser != 2 || tp.lastMsg.Type != KindDirectMessage || tp.lastMsg.From != "alice" || tp.lastMsg.Body != "hello" {
		t.Fatalf("unexpected transport call: to=%d msg=%+v", tp.toUser, tp.lastMsg)
	}
}

func TestRelayService_SendDirect_ReturnsOfflineReceiptWhenTransportFails(t *testing.T) {
	tp := &fakeTransport{ok: false}
	svc := NewRelayService(tp)

	receipt, err := svc.SendDirect(context.Background(), DirectSendRequest{
		From:      "alice",
		ToUserID:  2,
		Body:      "hello",
		MessageID: 45,
	})
	if err != nil {
		t.Fatalf("SendDirect returned error: %v", err)
	}
	if receipt.Delivered {
		t.Fatalf("expected undelivered receipt, got %+v", receipt)
	}
	if receipt.Reason != "recipient_offline" {
		t.Fatalf("unexpected reason %q", receipt.Reason)
	}
}
