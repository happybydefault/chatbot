package chatbot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/happybydefault/chatbot/data"
)

func (s *Server) handleChat(ctx context.Context, id string) error {
	chat, err := s.store.Chat(ctx, id)
	if err != nil {
		if err == data.ErrNotFound {
			s.logger.Warn("chat does not exist in the store", zap.String("chat_id", id))
			return nil
		}
		return fmt.Errorf("failed to get chat from store: %w", err)
	}
	s.logger.Debug("chat exists in the store", zap.String("chat_id", id))

	if len(chat.Messages) == 0 {
		return errors.New("chat has no messages")
	}

	jid, err := types.ParseJID(chat.ID)
	if err != nil {
		return fmt.Errorf("failed to parse JID: %w", err)
	}

	err = s.whatsmeow.SendChatPresence(jid, types.ChatPresenceComposing, "")
	if err != nil {
		return fmt.Errorf("failed to send chat composing presence: %w", err)
	}

	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()

	// TODO: Move prefix out of the function. Maybe use (Go) text templates.
	// TODO: Maybe encode conversation between the person and the AI as JSON.
	prefix := fmt.Sprintf(
		"The following is a conversation with an AI called Chatbot, the smartest of all beings."+
			" The assistant is helpful, creative, clever, and very friendly. The Chatbot's ID is %q.",
		"15024830330@s.whatsapp.net",
	)

	messages := make([]string, 0, len(chat.Messages))
	for _, message := range chat.Messages {
		messages = append(messages, fmt.Sprintf("%s:\n'''\n%s\n'''", message.SenderID, message.Text))
	}

	prompt := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		prefix,
		strings.Join(messages, "\n\n"),
		"15024830330@s.whatsapp.net:\n'''",
	)

	fmt.Printf("prompt:\n\n%s\n", prompt)

	completionResponse, err := s.completion(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to get completion response: %w", err)
	}
	if len(completionResponse.Choices) == 0 {
		return fmt.Errorf("received empty slice of completion choices")
	}

	conversationResponse := completionResponse.Choices[0].Text
	conversationResponse = strings.TrimSpace(conversationResponse)

	response := &waProto.Message{
		Conversation: proto.String(conversationResponse),
	}

	err = s.store.AppendMessage(ctx, id, data.Message{
		SenderID: "15024830330@s.whatsapp.net",
		Text:     conversationResponse,
	})
	if err != nil {
		return fmt.Errorf("failed to append chatbot's message to chat in store: %w", err)
	}

	// Make sure there is a delay between receiving a message and sending a response,
	// to avoid being tagged as a bot and getting banned.
	<-timer.C

	report, err := s.whatsmeow.SendMessage(ctx, jid, "", response)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	s.logger.Debug("message sent", zap.String("sent_message_id", report.ID))

	return nil
}
