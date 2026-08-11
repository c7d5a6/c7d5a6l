package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/api"
	"github.com/c7d5a6/c7d5a6l/internal/db"
	"github.com/c7d5a6/c7d5a6l/internal/debuglog"
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

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("POST /api/parse", api.ParseLink)

	addr := ":18765"
	log.Printf("backend listening on %s", addr)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		debuglog.Printf("http %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
