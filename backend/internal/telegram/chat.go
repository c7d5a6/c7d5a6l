package telegram

import "context"

type chatIDPayload struct {
	ChatID string `json:"chat_id"`
}

type chatMemberPayload struct {
	ChatID string `json:"chat_id"`
	UserID int64  `json:"user_id"`
}

// GetChat fetches the configured group (boot-time membership / id check).
func (c *Client) GetChat(ctx context.Context) (*Chat, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	var chat Chat
	if err := c.call(ctx, "getChat", chatIDPayload{ChatID: c.GroupID}, &chat); err != nil {
		return nil, err
	}
	return &chat, nil
}

// GetChatMember returns status for a known Telegram user in the configured group.
// The Bot API cannot list all members; this is a per-user check.
func (c *Client) GetChatMember(ctx context.Context, userID int64) (*ChatMember, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	var member ChatMember
	if err := c.call(ctx, "getChatMember", chatMemberPayload{ChatID: c.GroupID, UserID: userID}, &member); err != nil {
		return nil, err
	}
	return &member, nil
}
