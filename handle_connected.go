package chatbot

import (
	"fmt"

	"go.mau.fi/whatsmeow/types"
)

func (c *Client) handleConnectedEvent() error {
	c.status = StatusSyncing

	err := c.whatsmeowClient.SendPresence(types.PresenceAvailable)
	if err != nil {
		return fmt.Errorf("failed to send available presence: %w", err)
	}

	return nil
}
