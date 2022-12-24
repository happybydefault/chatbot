package chatbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PullRequestInc/go-gpt3"
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

	err := s.whatsapp.MarkRead(
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

	err = s.whatsapp.SendChatPresence(message.Info.Chat, types.ChatPresenceComposing, "")
	if err != nil {
		return fmt.Errorf("failed to send chat composing presence: %w", err)
	}

	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()

	// TODO: Move prefix out of the function. Maybe use (Go) text templates.
	// TODO: Maybe encode conversation between the person and the AI as JSON.
	prefix := "The following is a conversation with an AI called Chatbot, the smartest of all beings." +
		" The assistant is helpful, creative, clever, and very friendly." +
		"\n\n Person: "

	prompt := fmt.Sprintf(
		"%s\n\n%q\n\n%s:",
		prefix,
		message.Message.GetConversation(),
		"Chatbot:",
	)

	completionResponse, err := s.completion(ctx, prompt)
	if err != nil {
		return fmt.Errorf("failed to get completion response: %w", err)
	}
	if len(completionResponse.Choices) == 0 {
		return fmt.Errorf("received empty slice of completion choices")
	}
	s.logger.Debug(
		"received completion response",
		zap.String("completion_response", fmt.Sprintf("%#v", completionResponse)),
	)

	conversationResponse := completionResponse.Choices[0].Text
	conversationResponse = strings.TrimSpace(conversationResponse)

	response := &waProto.Message{
		Conversation: proto.String(conversationResponse),
	}

	// Make sure there is a delay between receiving a message and sending a response,
	// to avoid being tagged as a bot and getting banned.
	<-timer.C

	report, err := s.whatsapp.SendMessage(ctx, message.Info.Chat, "", response)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	s.logger.Debug("message sent", zap.String("sent_message_id", report.ID))

	return nil
}

func newCompletionRequest(prompts []string) gpt3.CompletionRequest {
	var (
		maxTokens           = 512
		temperature float32 = 0.0
	)

	completionRequest := gpt3.CompletionRequest{
		Prompt:      prompts,
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	}

	return completionRequest
}
