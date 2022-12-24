package chatbot

import (
	"context"
	"fmt"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/happybydefault/chatbot/data"
)

func (s *Server) handleMessage(ctx context.Context, message *events.Message) error {
	s.logger.Info(
		"Message event received",
		zap.String("message", message.Message.GetConversation()),
	)

	s.logger.Sugar().Debugf("sender: %#v", message.Info.Sender)

	err := s.client.MarkRead(
		[]types.MessageID{
			message.Info.ID,
		},
		time.Now(),
		message.Info.Chat,
		message.Info.Sender,
	)
	if err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	if message.Message.GetConversation() == "" {
		s.logger.Debug("ignoring Message event with empty Conversation")
		return nil
	}

	_, err = s.store.User(ctx, message.Info.Sender.User)
	if err != nil {
		if err == data.ErrNotFound {
			s.logger.Warn(
				"user does not exist in the store",
				zap.String("user_id", message.Info.Sender.User),
			)
			return nil
		}
		return fmt.Errorf("failed to get user from store: %w", err)
	}
	s.logger.Debug("user exists in the store")

	response := &waProto.Message{
		Conversation: proto.String(fmt.Sprintf(
			"Hello! You said: %q",
			message.Message.GetConversation(),
		)),
	}

	presenceTimer := time.NewTimer(time.Second)
	defer presenceTimer.Stop()

	responseTimer := time.NewTimer(3 * time.Second)
	defer responseTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-presenceTimer.C:
			err := s.client.SendChatPresence(message.Info.Chat, types.ChatPresenceComposing, "")
			if err != nil {
				return fmt.Errorf("failed to send chat composing presence: %w", err)
			}
		case <-responseTimer.C:
			report, err := s.client.SendMessage(ctx, message.Info.Chat, "", response)
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
			s.logger.Debug("message sent", zap.String("sent_message_id", report.ID))

			return nil
		}
	}
}
