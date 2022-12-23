package chatbot

import (
	"context"
	"fmt"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func (s *Server) handleMessage(ctx context.Context, message *events.Message) error {
	s.logger.Info(
		"message received",
		zap.String("message", message.Message.GetConversation()),
	)

	response := &waProto.Message{
		Conversation: proto.String(fmt.Sprintf(
			"message: %q\n\nevent: %#v",
			message.Message.GetConversation(),
			message,
		)),
	}

	_, err := s.client.SendMessage(ctx, message.Info.Chat, "", response)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
