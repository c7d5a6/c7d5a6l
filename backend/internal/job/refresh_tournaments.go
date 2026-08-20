package job

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/c7d5a6/c7d5a6l/internal/liquipedia"
	"github.com/c7d5a6/c7d5a6l/internal/repository"
	"github.com/c7d5a6/c7d5a6l/internal/service"
)

// Scheduler runs in-process cron jobs.
type Scheduler struct {
	cron *cron.Cron
}

// StartRefreshTournaments registers the 15-minute tournament refresh job and starts cron.
func StartRefreshTournaments(tours *service.Tournament, client *liquipedia.Client) *Scheduler {
	s := &Scheduler{cron: cron.New(cron.WithLocation(time.UTC))}
	job := &RefreshTournaments{Tours: tours, Client: client}
	_, err := s.cron.AddFunc("*/15 * * * *", func() {
		job.Run(context.Background())
	})
	if err != nil {
		log.Printf("job: schedule refresh tournaments: %v", err)
		return s
	}
	s.cron.Start()
	log.Printf("job: refresh tournaments scheduled every 15m (UTC)")
	return s
}

// Stop waits for running jobs then stops the scheduler.
func (s *Scheduler) Stop() context.Context {
	if s == nil || s.cron == nil {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}
	return s.cron.Stop()
}

// RefreshTournaments re-parses tournaments with past-due unplayed matches today.
type RefreshTournaments struct {
	Tours  *service.Tournament
	Client *liquipedia.Client

	mu sync.Mutex
}

// Run executes one refresh tick.
func (j *RefreshTournaments) Run(ctx context.Context) {
	if !j.mu.TryLock() {
		log.Printf("job refresh-tournaments: skip (overlap)")
		return
	}
	defer j.mu.Unlock()

	now := time.Now().UTC()
	log.Printf("job refresh-tournaments: tick start now=%s", now.Format(time.RFC3339))

	due, err := j.Tours.ListDueRefresh(ctx, now)
	if err != nil {
		log.Printf("job refresh-tournaments: list due: %v", err)
		return
	}
	log.Printf("job refresh-tournaments: candidates=%d", len(due))
	for _, t := range due {
		j.refreshOne(ctx, t)
	}
	log.Printf("job refresh-tournaments: tick done")
}

func (j *RefreshTournaments) refreshOne(ctx context.Context, t repository.TournamentDueRefresh) {
	log.Printf("job refresh-tournaments: refresh start id=%d link=%s", t.ID, t.Link)
	if j.Client == nil {
		log.Printf("job refresh-tournaments: refresh fail id=%d: liquipedia client nil", t.ID)
		return
	}
	fetched, err := j.Client.FetchPage(ctx, t.Link)
	if err != nil {
		log.Printf("job refresh-tournaments: refresh fail id=%d fetch: %v", t.ID, err)
		return
	}
	page, err := j.Tours.RefreshFromHTML(ctx, t.Link, fetched.HTML)
	if err != nil {
		log.Printf("job refresh-tournaments: refresh fail id=%d save: %v", t.ID, err)
		return
	}
	_ = page
	log.Printf("job refresh-tournaments: refresh ok id=%d results=%d groups=%d", t.ID, len(page.Results), len(page.Groups))
}