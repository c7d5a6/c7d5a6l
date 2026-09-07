package telegram

import "errors"

// ErrNotConfigured is returned when bot token or group id is missing.
var ErrNotConfigured = errors.New("telegram bot not configured")

// Chat is a Telegram chat from getChat.
type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// User is a Telegram user nested in ChatMember.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

// ChatMember is a getChatMember result. Status is creator, administrator,
// member, restricted, left, or kicked.
type ChatMember struct {
	Status string `json:"status"`
	User   User   `json:"user"`
}

// InChat reports whether the user currently belongs to the chat (including restricted).
func (m ChatMember) InChat() bool {
	switch m.Status {
	case "creator", "administrator", "member", "restricted":
		return true
	default:
		return false
	}
}
