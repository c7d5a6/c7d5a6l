package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/model"
)

// DefaultTelegramAuthMaxAge rejects Login Widget payloads older than this.
const DefaultTelegramAuthMaxAge = 24 * time.Hour

// VerifyTelegramAuth checks hash and auth_date per Telegram Login Widget docs.
func VerifyTelegramAuth(botToken string, payload model.TelegramAuthPayload, maxAge time.Duration) error {
	if botToken == "" {
		return fmt.Errorf("telegram bot token not configured")
	}
	if payload.ID == 0 || payload.FirstName == "" || payload.Hash == "" || payload.AuthDate == 0 {
		return fmt.Errorf("incomplete telegram auth payload")
	}
	if maxAge <= 0 {
		maxAge = DefaultTelegramAuthMaxAge
	}
	authTime := time.Unix(payload.AuthDate, 0)
	if time.Since(authTime) > maxAge || authTime.After(time.Now().Add(time.Minute)) {
		return fmt.Errorf("telegram auth_date out of range")
	}

	fields := map[string]string{
		"auth_date":  strconv.FormatInt(payload.AuthDate, 10),
		"first_name": payload.FirstName,
		"id":         strconv.FormatInt(payload.ID, 10),
	}
	if payload.LastName != "" {
		fields["last_name"] = payload.LastName
	}
	if payload.Username != "" {
		fields["username"] = payload.Username
	}
	if payload.PhotoURL != "" {
		fields["photo_url"] = payload.PhotoURL
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+fields[k])
	}
	dataCheck := strings.Join(lines, "\n")

	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	_, _ = mac.Write([]byte(dataCheck))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(strings.ToLower(payload.Hash))) {
		return fmt.Errorf("invalid telegram auth hash")
	}
	return nil
}
