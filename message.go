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
)

// TODO: This function is not exiting gracefully.
func (s *Server) handleMessage(ctx context.Context, message *events.Message) error {
	s.logger.Info(
		"Message event received",
		zap.String("message", message.Message.GetConversation()),
	)

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
			_, err := s.client.SendMessage(ctx, message.Info.Chat, "", response)
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
			return nil
		}
	}
}
