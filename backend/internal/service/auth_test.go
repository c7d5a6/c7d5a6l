package service_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/api"
	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/middleware"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

const testBotToken = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

func signTelegramPayload(botToken string, fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+fields[k])
	}
	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	_, _ = mac.Write([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(mac.Sum(nil))
}

func validPayload(t *testing.T, telegramID int64, firstName, username string) model.TelegramAuthPayload {
	t.Helper()
	authDate := time.Now().Unix()
	fields := map[string]string{
		"auth_date":  strconv.FormatInt(authDate, 10),
		"first_name": firstName,
		"id":         strconv.FormatInt(telegramID, 10),
	}
	if username != "" {
		fields["username"] = username
	}
	hash := signTelegramPayload(testBotToken, fields)
	return model.TelegramAuthPayload{
		ID:        telegramID,
		FirstName: firstName,
		Username:  username,
		AuthDate:  authDate,
		Hash:      hash,
	}
}

func TestVerifyTelegramAuth_okAndRejectsBadHash(t *testing.T) {
	p := validPayload(t, 42, "Jim", "raynor")
	if err := service.VerifyTelegramAuth(testBotToken, p, time.Hour); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
	p.Hash = "deadbeef"
	if err := service.VerifyTelegramAuth(testBotToken, p, time.Hour); err == nil {
		t.Fatal("expected bad hash error")
	}
}

func TestVerifyTelegramAuth_rejectsStaleAuthDate(t *testing.T) {
	p := validPayload(t, 7, "Old", "")
	p.AuthDate = time.Now().Add(-48 * time.Hour).Unix()
	fields := map[string]string{
		"auth_date":  strconv.FormatInt(p.AuthDate, 10),
		"first_name": p.FirstName,
		"id":         strconv.FormatInt(p.ID, 10),
	}
	p.Hash = signTelegramPayload(testBotToken, fields)
	if err := service.VerifyTelegramAuth(testBotToken, p, 24*time.Hour); err == nil {
		t.Fatal("expected stale auth_date error")
	}
}

func setupAuth(t *testing.T) (*service.Auth, *sql.DB, http.Handler) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.sqlite")
	sqlDB, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Migrate(context.Background(), sqlDB); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewUser(sqlDB)
	auth := service.NewAuth(sqlDB, repo, service.AuthConfig{
		BotToken:         testBotToken,
		BotID:            "123456",
		BotUsername:      "c7d5a6l_bot",
		JWTSecret:        "test-jwt-secret-please-change",
		JWTTTL:           time.Hour,
		AdminTelegramIDs: map[int64]struct{}{99: {}},
	})
	srv := &api.Server{Auth: auth}
	mux := http.NewServeMux()
	requireAuth := middleware.RequireAuth(auth)
	mux.HandleFunc("GET /api/auth/config", srv.AuthConfig)
	mux.HandleFunc("POST /api/auth/telegram", srv.AuthTelegram)
	mux.Handle("GET /api/me", requireAuth(http.HandlerFunc(srv.Me)))
	mux.Handle("PATCH /api/me", requireAuth(http.HandlerFunc(srv.PatchMe)))
	mux.Handle("POST /api/auth/logout", requireAuth(http.HandlerFunc(srv.AuthLogout)))
	mux.Handle("GET /api/users", requireAuth(middleware.RequireRole(model.RoleAdmin)(http.HandlerFunc(srv.ListUsers))))
	mux.Handle("POST /api/users", requireAuth(middleware.RequireRole(model.RoleAdmin)(http.HandlerFunc(srv.CreateUser))))
	return auth, sqlDB, mux
}

func TestAuthLoginUpsertAndAdminRole(t *testing.T) {
	auth, _, _ := setupAuth(t)
	ctx := context.Background()

	p := validPayload(t, 99, "Admin", "admin_user")
	token, user, err := auth.LoginTelegram(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	if user.Role != model.RoleAdmin {
		t.Fatalf("role=%s, want ADMIN", user.Role)
	}
	if user.Alias != "@admin_user" {
		t.Fatalf("alias=%s", user.Alias)
	}

	p2 := validPayload(t, 99, "Admin", "admin_user")
	_, user2, err := auth.LoginTelegram(ctx, p2)
	if err != nil {
		t.Fatal(err)
	}
	if user2.ID != user.ID {
		t.Fatalf("expected same user id, got %d vs %d", user2.ID, user.ID)
	}

	pUser := validPayload(t, 1001, "Sarah", "kerrigan")
	_, normal, err := auth.LoginTelegram(ctx, pUser)
	if err != nil {
		t.Fatal(err)
	}
	if normal.Role != model.RoleUser {
		t.Fatalf("role=%s, want USER", normal.Role)
	}
}

func TestLoginDoesNotOverwriteExistingAlias(t *testing.T) {
	auth, db, _ := setupAuth(t)
	ctx := context.Background()

	_, user, err := auth.LoginTelegram(ctx, validPayload(t, 501, "Raynor", "jimmy"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE user SET alias = ? WHERE id = ?`, "Marshal", user.ID); err != nil {
		t.Fatal(err)
	}

	_, again, err := auth.LoginTelegram(ctx, validPayload(t, 501, "Jim", "jimmy_new"))
	if err != nil {
		t.Fatal(err)
	}
	if again.Alias != "Marshal" {
		t.Fatalf("alias overwritten on login: got %q, want Marshal", again.Alias)
	}
}

func TestLoginFillsEmptyAlias(t *testing.T) {
	auth, db, _ := setupAuth(t)
	ctx := context.Background()

	_, user, err := auth.LoginTelegram(ctx, validPayload(t, 502, "Artanis", "artanis"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE user SET alias = '' WHERE id = ?`, user.ID); err != nil {
		t.Fatal(err)
	}

	_, again, err := auth.LoginTelegram(ctx, validPayload(t, 502, "Artanis", "artanis"))
	if err != nil {
		t.Fatal(err)
	}
	if again.Alias == "" {
		t.Fatal("expected empty alias to be filled on login")
	}
	if again.Alias != "@artanis" {
		t.Fatalf("alias=%q, want @artanis", again.Alias)
	}
}

func TestAPIMeRequiresBearer(t *testing.T) {
	auth, _, mux := setupAuth(t)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rec.Code)
	}

	p := validPayload(t, 55, "Nova", "nova")
	token, _, err := auth.LoginTelegram(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), `"alias":"@nova"`) {
		t.Fatalf("body=%s", rec2.Body.String())
	}
}

func TestAuthConfigExposesBotIDOnly(t *testing.T) {
	_, _, mux := setupAuth(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"botUsername":"c7d5a6l_bot"`) {
		t.Fatalf("body=%s", body)
	}
	if strings.Contains(body, testBotToken) || strings.Contains(strings.ToLower(body), "token") {
		t.Fatalf("leaked secret in body=%s", body)
	}
}

func TestLoginRejectsBadHash(t *testing.T) {
	auth, _, _ := setupAuth(t)
	p := validPayload(t, 1, "Bad", "hash")
	p.Hash = "00"
	_, _, err := auth.LoginTelegram(context.Background(), p)
	if err == nil {
		t.Fatal("expected error")
	}
	if fmt.Sprintf("%v", err) == "" {
		t.Fatal("empty error")
	}
}

func TestPatchMeAliasAndConflict(t *testing.T) {
	auth, _, mux := setupAuth(t)
	ctx := context.Background()

	tokenA, _, err := auth.LoginTelegram(ctx, validPayload(t, 201, "Alpha", "alpha"))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = auth.LoginTelegram(ctx, validPayload(t, 202, "Beta", "beta"))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/me", strings.NewReader(`{"alias":"Commander"}`))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"alias":"Commander"`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token":`) {
		t.Fatalf("missing token in body=%s", rec.Body.String())
	}

	tokenB, _, err := auth.LoginTelegram(ctx, validPayload(t, 202, "Beta", "beta"))
	if err != nil {
		t.Fatal(err)
	}
	req2 := httptest.NewRequest(http.MethodPatch, "/api/me", strings.NewReader(`{"alias":"Commander"}`))
	req2.Header.Set("Authorization", "Bearer "+tokenB)
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s want 409", rec2.Code, rec2.Body.String())
	}

	// Same alias for self is fine.
	req3 := httptest.NewRequest(http.MethodPatch, "/api/me", strings.NewReader(`{"alias":"Commander"}`))
	req3.Header.Set("Authorization", "Bearer "+tokenA)
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("self alias status=%d body=%s", rec3.Code, rec3.Body.String())
	}
}

func TestListUsersAdminOnly(t *testing.T) {
	auth, _, mux := setupAuth(t)
	ctx := context.Background()

	adminTok, _, err := auth.LoginTelegram(ctx, validPayload(t, 99, "Admin", "admin_user"))
	if err != nil {
		t.Fatal(err)
	}
	userTok, _, err := auth.LoginTelegram(ctx, validPayload(t, 1001, "Sarah", "kerrigan"))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no auth status=%d", rec.Code)
	}

	reqU := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	reqU.Header.Set("Authorization", "Bearer "+userTok)
	recU := httptest.NewRecorder()
	mux.ServeHTTP(recU, reqU)
	if recU.Code != http.StatusForbidden {
		t.Fatalf("user status=%d body=%s", recU.Code, recU.Body.String())
	}

	reqA := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	reqA.Header.Set("Authorization", "Bearer "+adminTok)
	recA := httptest.NewRecorder()
	mux.ServeHTTP(recA, reqA)
	if recA.Code != http.StatusOK {
		t.Fatalf("admin status=%d body=%s", recA.Code, recA.Body.String())
	}
	if !strings.Contains(recA.Body.String(), `"users":`) {
		t.Fatalf("body=%s", recA.Body.String())
	}
	if !strings.Contains(recA.Body.String(), "kerrigan") && !strings.Contains(recA.Body.String(), "@kerrigan") {
		t.Fatalf("expected users in body=%s", recA.Body.String())
	}
}

func TestCreateUserAliasOnly(t *testing.T) {
	auth, _, mux := setupAuth(t)
	ctx := context.Background()

	adminTok, _, err := auth.LoginTelegram(ctx, validPayload(t, 99, "Admin", "admin_user"))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"alias":"Ghost"}`))
	req.Header.Set("Authorization", "Bearer "+adminTok)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"alias":"Ghost"`) {
		t.Fatalf("body=%s", body)
	}
	if !strings.Contains(body, `"telegramId":null`) {
		t.Fatalf("expected null telegramId body=%s", body)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"alias":"Ghost"}`))
	req2.Header.Set("Authorization", "Bearer "+adminTok)
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", rec2.Code, rec2.Body.String())
	}

	userTok, _, err := auth.LoginTelegram(ctx, validPayload(t, 1001, "Sarah", "kerrigan"))
	if err != nil {
		t.Fatal(err)
	}
	req3 := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"alias":"Other"}`))
	req3.Header.Set("Authorization", "Bearer "+userTok)
	req3.Header.Set("Content-Type", "application/json")
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusForbidden {
		t.Fatalf("user create status=%d", rec3.Code)
	}
}

func TestAdminPOSTGates(t *testing.T) {
	auth, _, _ := setupAuth(t)
	ctx := context.Background()

	mux := http.NewServeMux()
	requireAuth := middleware.RequireAuth(auth)
	requireAdmin := func(h http.HandlerFunc) http.Handler {
		return requireAuth(middleware.RequireRole(model.RoleAdmin)(http.HandlerFunc(h)))
	}
	okHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}
	for _, path := range []string{
		"POST /api/parse",
		"POST /api/players",
		"POST /api/tournaments",
		"POST /api/fantasy-leagues",
	} {
		mux.Handle(path, requireAdmin(okHandler))
	}

	adminTok, _, err := auth.LoginTelegram(ctx, validPayload(t, 99, "Admin", "admin_user"))
	if err != nil {
		t.Fatal(err)
	}
	userTok, _, err := auth.LoginTelegram(ctx, validPayload(t, 1001, "Sarah", "kerrigan"))
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/api/parse", "/api/players", "/api/tournaments", "/api/fantasy-leagues"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s no auth status=%d", path, rec.Code)
		}

		reqU := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		reqU.Header.Set("Authorization", "Bearer "+userTok)
		recU := httptest.NewRecorder()
		mux.ServeHTTP(recU, reqU)
		if recU.Code != http.StatusForbidden {
			t.Fatalf("%s user status=%d", path, recU.Code)
		}

		reqA := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		reqA.Header.Set("Authorization", "Bearer "+adminTok)
		recA := httptest.NewRecorder()
		mux.ServeHTTP(recA, reqA)
		if recA.Code != http.StatusOK {
			t.Fatalf("%s admin status=%d body=%s", path, recA.Code, recA.Body.String())
		}
	}
}

