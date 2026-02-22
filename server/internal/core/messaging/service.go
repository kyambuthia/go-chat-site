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

// DurableRelayService persists direct messages and updates delivery receipts while
// preserving the same real-time relay semantics.
type DurableRelayService struct {
	transport   Transport
	persistence PersistenceService
}

func NewDurableRelayService(transport Transport, persistence PersistenceService) *DurableRelayService {
	return &DurableRelayService{
		transport:   transport,
		persistence: persistence,
	}
}

func (s *DurableRelayService) SendDirect(ctx context.Context, req DirectSendRequest) (DeliveryReceipt, error) {
	var storedID int64
	if s.persistence != nil {
		stored, err := s.persistence.StoreDirectMessage(ctx, PersistDirectMessageRequest{
			FromUserID: req.FromUserID,
			ToUserID:   req.ToUserID,
			Body:       req.Body,
		})
		if err != nil {
			return DeliveryReceipt{}, err
		}
		storedID = stored.ID
	}

	ok := s.transport.SendDirect(req.ToUserID, Message{
		Type: KindDirectMessage,
		From: req.From,
		Body: req.Body,
	})
	if !ok {
		return DeliveryReceipt{
			MessageID:       req.MessageID,
			StoredMessageID: storedID,
			Delivered:       false,
			Reason:          "recipient_offline",
		}, nil
	}

	if s.persistence != nil && storedID != 0 {
		if err := s.persistence.MarkDelivered(ctx, storedID); err != nil {
			return DeliveryReceipt{}, err
		}
	}

	return DeliveryReceipt{
		MessageID:       req.MessageID,
		StoredMessageID: storedID,
		Delivered:       true,
	}, nil
}
