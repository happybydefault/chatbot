package chatbot

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (s *Server) handleConnected(ctx context.Context, connected *events.Connected) error {
	err := s.client.SendPresence(types.PresenceAvailable)
	if err != nil {
		return fmt.Errorf("failed to send available presence: %w", err)
	}

	err = s.client.SetStatusMessage("👋")
	if err != nil {
		return fmt.Errorf("failed to set status message: %w", err)
	}

	return nil
}
