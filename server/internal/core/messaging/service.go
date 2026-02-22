package messaging

import "context"

type DirectSendRequest struct {
	FromUserID int
	From       string
	ToUserID   int
	Body       string
	MessageID  int64
}

type Service interface {
	SendDirect(ctx context.Context, req DirectSendRequest) (DeliveryReceipt, error)
}

// RelayService is a small core service used by centralized relay adapters today.
// Future P2P transports can satisfy the same messaging semantics with different adapters.
type RelayService struct {
	transport Transport
}

func NewRelayService(transport Transport) *RelayService {
	return &RelayService{transport: transport}
}

func (s *RelayService) SendDirect(ctx context.Context, req DirectSendRequest) (DeliveryReceipt, error) {
	_ = ctx
	ok := s.transport.SendDirect(req.ToUserID, Message{
		Type: KindDirectMessage,
		From: req.From,
		Body: req.Body,
	})
	if !ok {
		return DeliveryReceipt{MessageID: req.MessageID, Delivered: false, Reason: "recipient_offline"}, nil
	}
	return DeliveryReceipt{MessageID: req.MessageID, Delivered: true}, nil
}
