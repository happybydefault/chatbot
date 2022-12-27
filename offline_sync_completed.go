package chatbot

import (
	"context"
	"fmt"
	"sync"

	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

func (s *Server) handleOfflineSyncCompleted(
	ctx context.Context,
	offlineSyncCompleted *events.OfflineSyncCompleted,
) error {
	s.logger.Info(
		"OfflineSyncCompleted event received",
		zap.String("offline_sync_completed", fmt.Sprintf("%#v", offlineSyncCompleted)),
	)

	var wg sync.WaitGroup
	wg.Add(len(s.pendingChats))

	for chat := range s.pendingChats {
		chat := chat
		go func() {
			defer wg.Done()

			err := s.handleChat(ctx, chat.String())
			if err != nil {
				s.logger.Error("failed to handle chat", zap.Error(err))
			}
		}()
	}

	wg.Wait()
	s.state = StateReady

	return nil
}
