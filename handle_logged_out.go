package chatbot

import (
	"fmt"

	"go.mau.fi/whatsmeow/types/events"
)

// TODO: Revisit logic.
func (c *Client) handleLoggedOutEvent(loggedOut *events.LoggedOut) error {
	if !loggedOut.OnConnect {
		c.logger.Debug("whatsmeow client was logged out because of a stream:error message")
		return nil
	}

	if loggedOut.Reason.IsLoggedOut() {
		err := c.whatsmeowClient.Store.Delete()
		if err != nil {
			return fmt.Errorf("failed to delete store: %w", err)
		}
	}

	return fmt.Errorf("whatsmeow client was logged out because it failed to connect to WhatsApp")
}
