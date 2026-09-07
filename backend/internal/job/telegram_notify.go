package job

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/c7d5a6/c7d5a6l/internal/service"
)

const telegramNotifyInterval = 5 * time.Minute

// TelegramNotify ticks fantasy group messages.
type TelegramNotify struct {
	Notify *service.TelegramNotify

	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
}

// StartTelegramNotify runs an immediate tick then every 5 minutes.
func StartTelegramNotify(notify *service.TelegramNotify) *TelegramNotify {
	j := &TelegramNotify{
		Notify: notify,
		done:   make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	go j.loop(ctx)
	log.Printf("job: telegram notify scheduled every 5m")
	return j
}

// Stop cancels the worker and waits for the current tick to finish.
func (j *TelegramNotify) Stop() {
	if j == nil || j.cancel == nil {
		return
	}
	j.cancel()
	<-j.done
}

func (j *TelegramNotify) loop(ctx context.Context) {
	defer close(j.done)
	j.tick(ctx)
	ticker := time.NewTicker(telegramNotifyInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("job telegram-notify: stopped")
			return
		case <-ticker.C:
			j.tick(ctx)
		}
	}
}

func (j *TelegramNotify) tick(ctx context.Context) {
	if j.Notify == nil {
		return
	}
	if !j.mu.TryLock() {
		return
	}
	defer j.mu.Unlock()
	j.Notify.Tick(ctx, time.Now().UTC())
}
