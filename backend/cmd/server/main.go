package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/api"
	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
	"github.com/c7d5a6/c7d5a6l/internal/job"
	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/middleware"
	"github.com/c7d5a6/c7d5a6l/internal/model"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

func main() {
	debuglog.ConfigureFromEnv()
	if debuglog.Enabled() {
		log.Printf("debug logging enabled")
	}

	dbPath := db.ResolvePath(os.Getenv("C7D5A6L_DB"))
	debuglog.Printf("opening sqlite path=%s", dbPath)
	sqlDB, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("db open %s: %v", dbPath, err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(context.Background(), sqlDB); err != nil {
		log.Fatalf("db migrate: %v", err)
	}
	log.Printf("sqlite ready at %s", dbPath)

	lpClient := liquipedia.NewClient()
	playerRepo := repository.NewPlayer(sqlDB)
	tournamentRepo := repository.NewTournament(sqlDB)
	importRepo := repository.NewPlayerImport()
	playerFetcher := service.LiquipediaPlayerFetcher{Client: lpClient}
	playerSvc := service.NewPlayer(sqlDB, playerRepo, lpClient)
	tournamentSvc := service.NewTournament(
		sqlDB,
		tournamentRepo,
		playerRepo,
		importRepo,
		playerFetcher,
		lpClient,
	)
	playerImporter := service.NewPlayerImporter(sqlDB, importRepo, playerRepo, playerFetcher, lpClient)
	fantasyRepo := repository.NewFantasy(sqlDB)
	seasonRepo := repository.NewSeason(sqlDB)
	seasonSvc := service.NewSeason(sqlDB, seasonRepo, playerRepo)
	fantasySvc := service.NewFantasy(sqlDB, fantasyRepo, tournamentRepo, seasonSvc)
	userRepo := repository.NewUser(sqlDB)
	titleRepo := repository.NewTitle(sqlDB)
	titleSvc := service.NewTitle(sqlDB, titleRepo, userRepo, fantasyRepo)
	authSvc := service.NewAuth(sqlDB, userRepo, service.AuthConfig{
		BotToken:         os.Getenv("C7D5A6L_TELEGRAM_BOT_TOKEN"),
		BotID:            os.Getenv("C7D5A6L_TELEGRAM_BOT_ID"),
		BotUsername:      os.Getenv("C7D5A6L_TELEGRAM_BOT_USERNAME"),
		JWTSecret:        os.Getenv("C7D5A6L_JWT_SECRET"),
		JWTTTL:           service.ParseDurationEnv(os.Getenv("C7D5A6L_JWT_TTL"), 7*24*time.Hour),
		AdminTelegramIDs: service.ParseAdminTelegramIDs(os.Getenv("C7D5A6L_ADMIN_TELEGRAM_IDS")),
	})
	if !authSvc.Configured() {
		log.Printf("auth: Telegram login disabled (set C7D5A6L_TELEGRAM_BOT_TOKEN, C7D5A6L_TELEGRAM_BOT_USERNAME, C7D5A6L_JWT_SECRET)")
	}
	apiServer := &api.Server{
		Liquipedia:  lpClient,
		Players:     playerSvc,
		Tournaments: tournamentSvc,
		Fantasy:     fantasySvc,
		Seasons:     seasonSvc,
		Auth:        authSvc,
		Titles:      titleSvc,
	}

	mux := http.NewServeMux()
	requireAuth := middleware.RequireAuth(authSvc)
	requireAdmin := func(h http.HandlerFunc) http.Handler {
		return requireAuth(middleware.RequireRole(model.RoleAdmin)(http.HandlerFunc(h)))
	}

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.Handle("POST /api/parse", requireAdmin(apiServer.ParseLink))
	mux.HandleFunc("GET /api/players", apiServer.ListPlayers)
	mux.Handle("POST /api/players", requireAdmin(apiServer.SavePlayer))
	mux.Handle("PATCH /api/players/races/{id}", requireAdmin(apiServer.PatchPlayerRace))
	mux.HandleFunc("GET /api/players/lookup", apiServer.GetPlayerLookup)
	mux.HandleFunc("GET /api/players/portrait", apiServer.GetPlayerPortrait)
	mux.HandleFunc("GET /api/tournaments", apiServer.ListTournaments)
	mux.Handle("GET /api/tournaments/unused-for-fantasy", requireAdmin(apiServer.ListUnusedTournamentsForFantasy))
	mux.Handle("GET /api/tournaments/{id}", requireAdmin(apiServer.GetTournament))
	mux.Handle("POST /api/tournaments", requireAdmin(apiServer.SaveTournament))
	mux.Handle("GET /api/tournament-queue", requireAdmin(apiServer.ListAdminTournaments))
	mux.Handle("POST /api/tournament-queue/sync", requireAdmin(apiServer.SyncTournamentQueue))
	mux.Handle("POST /api/tournament-queue/{id}/parse", requireAdmin(apiServer.ParseTournamentQueue))
	mux.Handle("POST /api/tournament-queue/{id}/ignore", requireAdmin(apiServer.IgnoreTournamentQueue))

	mux.Handle("GET /api/seasons/current", requireAuth(http.HandlerFunc(apiServer.GetCurrentSeason)))
	mux.Handle("GET /api/seasons/close-preview", requireAdmin(apiServer.GetSeasonClosePreview))
	mux.Handle("POST /api/seasons/close", requireAdmin(apiServer.CloseSeason))

	mux.HandleFunc("GET /api/fantasy-leagues", apiServer.ListFantasyLeagues)
	mux.HandleFunc("GET /api/fantasy-leagues/active", apiServer.GetActiveFantasyLeague)
	mux.Handle("GET /api/fantasy-leagues/preview", requireAdmin(apiServer.PreviewFantasyLeague))
	mux.Handle("POST /api/fantasy-leagues", requireAdmin(apiServer.CreateFantasyLeague))
	mux.HandleFunc("GET /api/fantasy-leagues/{id}", apiServer.GetFantasyLeague)
	mux.Handle("PATCH /api/fantasy-leagues/{id}", requireAdmin(apiServer.PatchFantasyLeague))
	mux.Handle("POST /api/fantasy-leagues/{id}/start", requireAdmin(apiServer.StartFantasyLeague))
	mux.Handle("POST /api/fantasy-leagues/{id}/finish", requireAdmin(apiServer.FinishFantasyLeague))
	mux.HandleFunc("GET /api/fantasy-leagues/{id}/players", apiServer.ListFantasyPlayers)
	mux.Handle("PATCH /api/fantasy-leagues/{id}/players/{playerId}", requireAdmin(apiServer.PatchFantasyPlayer))
	mux.HandleFunc("GET /api/fantasy-leagues/{id}/groups", apiServer.ListFantasyGroups)
	mux.HandleFunc("GET /api/fantasy-leagues/{id}/match-board", apiServer.GetFantasyMatchBoard)
	mux.HandleFunc("GET /api/fantasy-leagues/{id}/teams", apiServer.ListFantasyTeams)
	mux.Handle("POST /api/fantasy-leagues/{id}/teams", requireAdmin(apiServer.CreateFantasyTeam))
	mux.Handle("PUT /api/fantasy-leagues/{id}/teams/{teamId}", requireAdmin(apiServer.UpdateFantasyTeam))
	mux.Handle("DELETE /api/fantasy-leagues/{id}/teams/{teamId}", requireAdmin(apiServer.DeleteFantasyTeam))
	mux.Handle("GET /api/fantasy-leagues/{id}/my-team", requireAuth(http.HandlerFunc(apiServer.GetMyFantasyTeam)))
	mux.Handle("PUT /api/fantasy-leagues/{id}/my-team", requireAuth(http.HandlerFunc(apiServer.PutMyFantasyTeam)))

	mux.HandleFunc("GET /api/auth/config", apiServer.AuthConfig)
	mux.HandleFunc("POST /api/auth/telegram", apiServer.AuthTelegram)
	mux.Handle("GET /api/me", requireAuth(http.HandlerFunc(apiServer.Me)))
	mux.Handle("PATCH /api/me", requireAuth(http.HandlerFunc(apiServer.PatchMe)))
	mux.Handle("POST /api/auth/logout", requireAuth(http.HandlerFunc(apiServer.AuthLogout)))
	mux.Handle("GET /api/users", requireAdmin(apiServer.ListUsers))
	mux.Handle("POST /api/users", requireAdmin(apiServer.CreateUser))
	mux.Handle("PATCH /api/users/{id}", requireAdmin(apiServer.UpdateUser))
	mux.HandleFunc("GET /api/user-titles", apiServer.ListUserTitles)
	mux.Handle("POST /api/user-titles", requireAdmin(apiServer.CreateUserTitle))
	mux.HandleFunc("GET /api/user-titles/{id}/image", apiServer.GetUserTitleImage)
	mux.Handle("PATCH /api/user-titles/{id}", requireAdmin(apiServer.UpdateUserTitle))
	mux.Handle("DELETE /api/user-titles/{id}", requireAdmin(apiServer.DeleteUserTitle))

	sched := job.StartRefreshTournaments(tournamentSvc, lpClient)
	importer := job.StartImportPlayers(playerImporter)

	addr := ":18765"
	srv := &http.Server{Addr: addr, Handler: withCORS(mux)}
	go func() {
		log.Printf("backend listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("shutdown signal=%v", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	importer.Stop()
	<-sched.Stop().Done()
	log.Printf("shutdown complete")
}

func withCORS(next http.Handler) http.Handler {
	allowed := map[string]struct{}{
		"http://localhost:3000":     {},
		"http://127.0.0.1:3000":     {},
		"https://c7d5a6l.lo":        {},
		"https://league.c7d5a6.com": {},
	}
	if extra := strings.TrimSpace(os.Getenv("C7D5A6L_CORS_ORIGIN")); extra != "" {
		for _, o := range strings.Split(extra, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowed[o] = struct{}{}
			}
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Set("Vary", "Origin")
		if _, ok := allowed[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		debuglog.Printf("http %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
