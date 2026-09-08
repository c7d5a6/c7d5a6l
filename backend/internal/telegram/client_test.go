package telegram_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c7d5a6/c7d5a6l/internal/telegram"
)

const testToken = "123456:TESTTOKEN"
const testGroup = "-1001234567890"

func TestConfigured(t *testing.T) {
	t.Parallel()

	if telegram.New("", testGroup).Configured() {
		t.Fatal("expected unconfigured without token")
	}
	if telegram.New(testToken, "").Configured() {
		t.Fatal("expected unconfigured without group id")
	}
	if telegram.New("  ", "  ").Configured() {
		t.Fatal("expected unconfigured after trim of blanks")
	}
	if !telegram.New(" "+testToken+" ", " "+testGroup+" ").Configured() {
		t.Fatal("expected configured after trim")
	}
}

func TestSendGroup_notConfigured(t *testing.T) {
	t.Parallel()

	err := telegram.New("", testGroup).SendGroup(context.Background(), "hello world")
	if !errors.Is(err, telegram.ErrNotConfigured) {
		t.Fatalf("got %v, want ErrNotConfigured", err)
	}
}

func TestSendGroup_ok(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			t.Errorf("path %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.Path, "/bot"+testToken+"/") {
			t.Errorf("token missing from path %s", r.URL.Path)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		var payload map[string]string
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Errorf("json: %v", err)
		}
		if payload["chat_id"] != testGroup {
			t.Errorf("chat_id %q", payload["chat_id"])
		}
		if payload["text"] != "hello world" {
			t.Errorf("text %q", payload["text"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	defer srv.Close()

	c := telegram.New(testToken, testGroup)
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()

	if err := c.SendGroup(context.Background(), "hello world"); err != nil {
		t.Fatalf("SendGroup: %v", err)
	}
}

func TestSendGroup_apiError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":403,"description":"Forbidden: bot is not a member of the group chat"}`))
	}))
	defer srv.Close()

	c := telegram.New(testToken, testGroup)
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()

	err := c.SendGroup(context.Background(), "hello world")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Forbidden") {
		t.Fatalf("error %v", err)
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("error %v should include code", err)
	}
}

func TestGetChatMember(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/getChatMember") {
			t.Errorf("path %s", r.URL.Path)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Errorf("json: %v", err)
		}
		if payload["chat_id"] != testGroup {
			t.Errorf("chat_id %#v", payload["chat_id"])
		}
		if payload["user_id"] != float64(4242) {
			t.Errorf("user_id %#v", payload["user_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"status":"member","user":{"id":4242,"is_bot":false,"first_name":"Jim","username":"raynor"}}}`))
	}))
	defer srv.Close()

	c := telegram.New(testToken, testGroup)
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()

	m, err := c.GetChatMember(context.Background(), 4242)
	if err != nil {
		t.Fatalf("GetChatMember: %v", err)
	}
	if m.Status != "member" || !m.InChat() {
		t.Fatalf("member %+v", m)
	}
	if m.User.ID != 4242 || m.User.Username != "raynor" {
		t.Fatalf("user %+v", m.User)
	}
}

func TestGetChatMember_left(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"status":"left","user":{"id":7,"is_bot":false,"first_name":"X"}}}`))
	}))
	defer srv.Close()

	c := telegram.New(testToken, testGroup)
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()

	m, err := c.GetChatMember(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetChatMember: %v", err)
	}
	if m.InChat() {
		t.Fatalf("left should not be InChat: %+v", m)
	}
}

func TestGetChatMember_notConfigured(t *testing.T) {
	t.Parallel()

	_, err := telegram.New(testToken, "").GetChatMember(context.Background(), 1)
	if !errors.Is(err, telegram.ErrNotConfigured) {
		t.Fatalf("got %v, want ErrNotConfigured", err)
	}
}

func TestGetChat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/getChat") {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"id":-1001234567890,"type":"supergroup","title":"ASL"}}`))
	}))
	defer srv.Close()

	c := telegram.New(testToken, testGroup)
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()

	chat, err := c.GetChat(context.Background())
	if err != nil {
		t.Fatalf("GetChat: %v", err)
	}
	if chat.ID != -1001234567890 || chat.Type != "supergroup" || chat.Title != "ASL" {
		t.Fatalf("chat %+v", chat)
	}
}
