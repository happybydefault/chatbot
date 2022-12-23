package chatbot

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// TODO: Revisit logic.
func (s *Server) handleLoggedOut(ctx context.Context, loggedOut *events.LoggedOut) error {
	s.logger.Info(
		"LoggedOut event received",
		zap.String("qr_codes", fmt.Sprintf("%#v", loggedOut)),
	)

	if loggedOut.OnConnect {
		if loggedOut.Reason.IsLoggedOut() {
			err := s.client.Store.Delete()
			if err != nil {
				return fmt.Errorf("failed to delete store: %w", err)
			}
		}

		return fmt.Errorf("client was logged out because it failed to connect to WhatsApp")
	}

	return nil
}
