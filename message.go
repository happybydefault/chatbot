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

func (s *Server) handleMessage(ctx context.Context, message *events.Message) error {
	s.logger.Info(
		"message received",
		zap.String("message", message.Message.GetConversation()),
	)

	time.Sleep(1 * time.Second)
	err := s.client.SendChatPresence(message.Info.Chat, types.ChatPresenceComposing, "")
	if err != nil {
		return fmt.Errorf("failed to send chat composing presence: %w", err)
	}

	time.Sleep(3 * time.Second)
	response := &waProto.Message{
		Conversation: proto.String(fmt.Sprintf(
			"Hello! You said: %q",
			message.Message.GetConversation(),
		)),
	}

	_, err = s.client.SendMessage(ctx, message.Info.Chat, "", response)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
