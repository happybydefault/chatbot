package chatbot

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

func (s *Server) handleOfflineSyncPreview(ctx context.Context, offlineSyncPreview *events.OfflineSyncPreview) error {
	s.logger.Info(
		"OfflineSyncPreview event received",
		zap.String("offline_sync_preview", fmt.Sprintf("%#v", offlineSyncPreview)),
	)

	s.state = StateSyncing

	return nil
}
