package chatbot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PullRequestInc/go-gpt3"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"

	"github.com/happybydefault/chatbot/data"
)

func (s *Server) handleMessage(ctx context.Context, message *events.Message) {
	s.logger.Info(
		"Message event received",
		zap.String("message", fmt.Sprintf("%#v", message)),
	)

	if message.Message.GetConversation() == "" {
		s.logger.Debug("ignoring Message event with empty Conversation")
		return
	}

	chatID := message.Info.Chat.ToNonAD().String()

	_, err := s.store.Chat(ctx, chatID)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			s.logger.Warn("chat does not exist in the store", zap.String("chat_id", chatID))
		} else {
			s.logger.Error("failed to get chat from the store", zap.Error(err))
		}
		return
	}
	s.logger.Debug("chat exists in the store", zap.String("chat_id", chatID))

	err = s.store.AppendMessage(ctx, chatID, data.Message{
		SenderID: message.Info.Sender.ToNonAD().String(),
		Text:     message.Message.GetConversation(),
	})
	if err != nil {
		s.logger.Error("failed to append user's message to the chat in the store", zap.Error(err))
		return
	}

	err = s.whatsmeow.MarkRead(
		[]types.MessageID{
			message.Info.ID,
		},
		time.Now(),
		message.Info.Chat.ToNonAD(),
		message.Info.Sender.ToNonAD(),
	)
	if err != nil {
		s.logger.Error("failed to mark message as read", zap.Error(err))
		return
	}

	if s.state == StateSyncing {
		s.logger.Debug("adding chat to the pending chats", zap.String("chat_id", chatID))
		s.pendingChats[message.Info.Chat] = struct{}{}
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		err = s.handleChat(ctx, message.Info.Chat.String())
		if err != nil {
			s.logger.Error("failed to handle chat", zap.Error(err))
		}
	}()
}

func newCompletionRequest(prompts []string) gpt3.CompletionRequest {
	var (
		maxTokens           = 512
		temperature float32 = 0.0
		stop                = []string{"'''"}
	)

	completionRequest := gpt3.CompletionRequest{
		Prompt:      prompts,
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
		Stop:        stop,
	}

	return completionRequest
}
