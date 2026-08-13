package model

// User roles.
const (
	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

// User is an authenticated account (Telegram Login and/or admin-created alias).
type User struct {
	ID               int64       `json:"id"`
	Alias            string      `json:"alias"`
	TelegramID       *int64      `json:"telegramId"`
	TelegramUsername *string     `json:"telegramUsername"`
	FirstName        string      `json:"firstName"`
	LastName         *string     `json:"lastName"`
	PhotoURL         *string     `json:"photoUrl"`
	Role             string      `json:"role"`
	CreatedAt        string      `json:"createdAt"`
	UpdatedAt        string      `json:"updatedAt"`
	LastLoginAt      *string     `json:"lastLoginAt"`
	Titles           []UserTitle `json:"titles,omitempty"`
}

// TelegramAuthPayload is the Login Widget callback object (verified server-side).
type TelegramAuthPayload struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}
