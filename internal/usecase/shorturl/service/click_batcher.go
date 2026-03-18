package service

import (
	"context"
	"sync"
	"time"

	"github.com/gattolab/wrappp/internal/domain"
	"github.com/gattolab/wrappp/pkg/logger"
)

const (
	clickChannelSize = 16384       // buffer up to 16k pending clicks before dropping
	flushInterval    = time.Second // flush to DB every second
	flushBatchSize   = 500         // also flush early when batch reaches this size
)

// ClickBatcher aggregates click increments in memory and flushes them
// to the DB in batches. This lets 10k+ concurrent requests each send
// one non-blocking channel write instead of spawning a goroutine or
// hitting the DB directly.
type ClickBatcher struct {
	ch     chan string
	repo   domain.ShortUrlRepository
	logger logger.Logger
	once   sync.Once
	stopCh chan struct{}
	doneCh chan struct{}
}

func NewClickBatcher(repo domain.ShortUrlRepository, log logger.Logger) *ClickBatcher {
	b := &ClickBatcher{
		ch:     make(chan string, clickChannelSize),
		repo:   repo,
		logger: log,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	go b.run()
	return b
}

// Record enqueues a click for the given code. Non-blocking: drops the
// event if the buffer is full (back-pressure) rather than stalling the
// HTTP response.
func (b *ClickBatcher) Record(code string) {
	select {
	case b.ch <- code:
	default:
		// channel full — drop rather than block the caller
	}
}

// Stop signals the worker to drain and flush, then waits until it exits.
func (b *ClickBatcher) Stop() {
	b.once.Do(func() { close(b.stopCh) })
	<-b.doneCh
}

func (b *ClickBatcher) run() {
	defer close(b.doneCh)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	counts := make(map[string]int64)

	flush := func() {
		if len(counts) == 0 {
			return
		}
		ctx := context.Background()
		for code, n := range counts {
			if err := b.repo.IncrementClickBy(ctx, code, n); err != nil {
				b.logger.Error(ctx, "click batcher: failed to flush clicks for "+code+": "+err.Error())
			}
		}
		counts = make(map[string]int64)
	}

	for {
		select {
		case code := <-b.ch:
			counts[code]++
			if len(counts) >= flushBatchSize {
				flush()
			}

		case <-ticker.C:
			flush()

		case <-b.stopCh:
			// drain remaining items in channel before exit
			for {
				select {
				case code := <-b.ch:
					counts[code]++
				default:
					flush()
					return
				}
			}
		}
	}
}
