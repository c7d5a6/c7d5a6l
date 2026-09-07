package telegram

import "context"

type sendMessagePayload struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

// SendMessage posts text to chatID via sendMessage.
func (c *Client) SendMessage(ctx context.Context, chatID, text string) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	return c.call(ctx, "sendMessage", sendMessagePayload{ChatID: chatID, Text: text}, nil)
}

// SendGroup posts text to the configured group.
func (c *Client) SendGroup(ctx context.Context, text string) error {
	if !c.Configured() {
		return ErrNotConfigured
	}
	return c.SendMessage(ctx, c.GroupID, text)
}
