package chatbot

func (c *Client) handleOfflineSyncCompletedEvent() error {
	c.state = StateSynced

	return nil
}
