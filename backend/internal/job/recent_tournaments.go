package job

import (
	"context"
	"log"
	"sync"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

// RecentTournaments scrapes Liquipedia Recent Tournaments into the admin queue.
type RecentTournaments struct {
	Tours  *service.Tournament
	Client *liquipedia.Client

	mu sync.Mutex
}

// Run fetches the listing page and upserts the tournament queue.
func (j *RecentTournaments) Run(ctx context.Context) {
	if !j.mu.TryLock() {
		log.Printf("job recent-tournaments: skip (overlap)")
		return
	}
	defer j.mu.Unlock()

	log.Printf("job recent-tournaments: start")
	if j.Client == nil {
		log.Printf("job recent-tournaments: liquipedia client nil")
		return
	}
	if j.Tours == nil {
		log.Printf("job recent-tournaments: tournament service nil")
		return
	}
	fetched, err := j.Client.FetchPage(ctx, liquipedia.RecentTournamentsURL)
	if err != nil {
		log.Printf("job recent-tournaments: fetch: %v", err)
		return
	}
	n, err := j.Tours.SyncRecentFromHTML(ctx, fetched.HTML)
	if err != nil {
		log.Printf("job recent-tournaments: save: %v", err)
		return
	}
	log.Printf("job recent-tournaments: ok listings=%d", n)
}
