package job

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/service"
)

// ImportPlayers drains the player_import_queue one Liquipedia page at a time.
type ImportPlayers struct {
	Importer *service.PlayerImporter

	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
}

// StartImportPlayers recovers interrupted jobs and processes the queue in the background.
func StartImportPlayers(importer *service.PlayerImporter) *ImportPlayers {
	j := &ImportPlayers{
		Importer: importer,
		done:     make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	go j.loop(ctx)
	log.Printf("job: player import worker started")
	return j
}

// Stop cancels the worker and waits for the current tick to finish.
func (j *ImportPlayers) Stop() {
	if j == nil || j.cancel == nil {
		return
	}
	j.cancel()
	<-j.done
}

func (j *ImportPlayers) loop(ctx context.Context) {
	defer close(j.done)
	if j.Importer != nil {
		if n, err := j.Importer.RecoverRunning(ctx); err != nil {
			log.Printf("job player-import: recover: %v", err)
		} else if n > 0 {
			log.Printf("job player-import: re-queued %d interrupted job(s)", n)
		}
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("job player-import: stopped")
			return
		case <-ticker.C:
			j.tick(ctx)
		}
	}
}

func (j *ImportPlayers) tick(ctx context.Context) {
	if j.Importer == nil {
		return
	}
	if !j.mu.TryLock() {
		return
	}
	defer j.mu.Unlock()

	link, err := j.Importer.ProcessNext(ctx)
	if err != nil {
		log.Printf("job player-import: process: %v", err)
		return
	}
	if link != "" {
		log.Printf("job player-import: enriched %s", link)
	}
}
