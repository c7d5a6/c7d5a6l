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

// StartRefreshTournaments registers refresh jobs and runs unfinished tournaments once on boot.
func StartRefreshTournaments(tours *service.Tournament, client *liquipedia.Client) *Scheduler {
	s := &Scheduler{cron: cron.New(cron.WithLocation(time.UTC))}
	job := &RefreshTournaments{Tours: tours, Client: client}
	if _, err := s.cron.AddFunc("*/15 * * * *", func() {
		job.RunDue(context.Background())
	}); err != nil {
		log.Printf("job: schedule refresh tournaments 15m: %v", err)
		return s
	}
	if _, err := s.cron.AddFunc("*/2 * * * *", func() {
		job.RunInProgress(context.Background())
	}); err != nil {
		log.Printf("job: schedule refresh tournaments 2m: %v", err)
		return s
	}
	s.cron.Start()
	log.Printf("job: refresh tournaments scheduled every 15m (due) and 2m (in-progress) UTC")
	go job.RunUnfinished(context.Background())
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

// RefreshTournaments re-parses tournaments from Liquipedia.
type RefreshTournaments struct {
	Tours  *service.Tournament
	Client *liquipedia.Client

	mu sync.Mutex
}

// RunDue executes the 15-minute due-today refresh tick.
func (j *RefreshTournaments) RunDue(ctx context.Context) {
	j.runListed(ctx, "due", func() ([]repository.TournamentDueRefresh, error) {
		return j.Tours.ListDueRefresh(ctx, time.Now().UTC())
	})
}

// RunInProgress executes the 2-minute in-progress refresh tick.
func (j *RefreshTournaments) RunInProgress(ctx context.Context) {
	j.runListed(ctx, "in-progress", func() ([]repository.TournamentDueRefresh, error) {
		return j.Tours.ListInProgressRefresh(ctx, time.Now().UTC())
	})
}

// RunUnfinished refreshes all unfinished tournaments (used on boot).
func (j *RefreshTournaments) RunUnfinished(ctx context.Context) {
	j.runListed(ctx, "unfinished", func() ([]repository.TournamentDueRefresh, error) {
		return j.Tours.ListUnfinished(ctx)
	})
}

func (j *RefreshTournaments) runListed(ctx context.Context, kind string, list func() ([]repository.TournamentDueRefresh, error)) {
	if !j.mu.TryLock() {
		log.Printf("job refresh-tournaments: skip %s (overlap)", kind)
		return
	}
	defer j.mu.Unlock()

	now := time.Now().UTC()
	log.Printf("job refresh-tournaments: %s start now=%s", kind, now.Format(time.RFC3339))

	due, err := list()
	if err != nil {
		log.Printf("job refresh-tournaments: list %s: %v", kind, err)
		return
	}
	log.Printf("job refresh-tournaments: %s candidates=%d", kind, len(due))
	for _, t := range due {
		j.refreshOne(ctx, t)
	}
	log.Printf("job refresh-tournaments: %s done", kind)
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
	log.Printf("job refresh-tournaments: refresh ok id=%d results=%d groups=%d", t.ID, len(page.Results), len(page.Groups))
}
