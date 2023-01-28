package chatbot

import (
	"go.mau.fi/whatsmeow/types/events"
)

func (c *Client) handleMessageEvent(msg *events.Message) {
	chatID := msg.Info.Chat.User
	chat := c.getChat(chatID)

	chat.messagesChan <- message{
		clientState: c.state,
		Message:     msg,
	}
}

func (c *Client) getChat(chatID string) *Chat {
	c.mu.Lock()

	chat, ok := c.chats[chatID]
	if !ok {
		chat = c.newChat(chatID)
		c.chats[chatID] = chat

		go chat.handleMessages()
	}

	c.mu.Unlock()

	return chat
}
