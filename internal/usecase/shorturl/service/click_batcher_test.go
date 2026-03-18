package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gormLogger "gorm.io/gorm/logger"

	"github.com/gattolab/wrappp/internal/domain"
	"github.com/gattolab/wrappp/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"

	zaplib "go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Mock logger
// ---------------------------------------------------------------------------

type mockLogger struct{}

func (m *mockLogger) InitLogger()                                         {}
func (m *mockLogger) Debugf(_ string, _ ...interface{})                   {}
func (m *mockLogger) Infof(_ string, _ ...interface{})                    {}
func (m *mockLogger) Warnf(_ string, _ ...interface{})                    {}
func (m *mockLogger) Errorf(_ string, _ ...interface{})                   {}
func (m *mockLogger) DPanicf(_ string, _ ...interface{})                  {}
func (m *mockLogger) Panicf(_ string, _ ...interface{})                   {}
func (m *mockLogger) Fatalf(_ string, _ ...interface{})                   {}
func (m *mockLogger) WithFiled(_ zapcore.Field) *zaplib.Logger            { return nil }
func (m *mockLogger) LogMode(_ gormLogger.LogLevel) gormLogger.Interface  { return m }
func (m *mockLogger) Info(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLogger) Warn(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLogger) Error(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) Trace(_ context.Context, _ time.Time, _ func() (string, int64), _ error) {
}

// ---------------------------------------------------------------------------
// Mock repository — only IncrementClickBy matters here
// ---------------------------------------------------------------------------

type mockRepo struct {
	mu            sync.Mutex
	callCount     int64 // number of IncrementClickBy calls
	totalReceived int64 // sum of all n values passed
	perCode       map[string]int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{perCode: make(map[string]int64)}
}

func (r *mockRepo) IncrementClickBy(_ context.Context, code string, n int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	atomic.AddInt64(&r.callCount, 1)
	atomic.AddInt64(&r.totalReceived, n)
	r.perCode[code] += n
	return nil
}

func (r *mockRepo) IncrementClick(_ context.Context, _ string) error { return nil }
func (r *mockRepo) Create(_ context.Context, s entity.ShortUrl) (entity.ShortUrl, error) {
	return s, nil
}
func (r *mockRepo) Update(_ context.Context, s *entity.ShortUrl) (*entity.ShortUrl, error) {
	return s, nil
}
func (r *mockRepo) GetByCode(_ context.Context, _ string) (*entity.ShortUrl, error) { return nil, nil }
func (r *mockRepo) GetAll(_ context.Context) ([]entity.ShortUrl, error)             { return nil, nil }
func (r *mockRepo) DeleteByCode(_ context.Context, _ string) error                  { return nil }

// Compile-time check that mockRepo satisfies the interface.
var _ domain.ShortUrlRepository = (*mockRepo)(nil)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestClickBatcher_ConcurrentRecords sends 10 000 concurrent clicks for
// the same code and verifies every single one is flushed to the DB.
func TestClickBatcher_ConcurrentRecords(t *testing.T) {
	repo := newMockRepo()
	batcher := NewClickBatcher(repo, &mockLogger{})

	const total = 10_000
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		go func() {
			defer wg.Done()
			batcher.Record("abc123")
		}()
	}
	wg.Wait()

	// Stop drains and flushes everything before returning.
	batcher.Stop()

	repo.mu.Lock()
	got := repo.perCode["abc123"]
	repo.mu.Unlock()

	assert.Equal(t, int64(total), got,
		"all %d clicks must be flushed to the DB", total)
}

// TestClickBatcher_MultipleCodesAccumulate verifies that counts are
// aggregated per code and each code gets the right total.
func TestClickBatcher_MultipleCodesAccumulate(t *testing.T) {
	repo := newMockRepo()
	batcher := NewClickBatcher(repo, &mockLogger{})

	codes := []string{"aaa", "bbb", "ccc"}
	const clicksPerCode = 3_000

	var wg sync.WaitGroup
	for _, code := range codes {
		for i := 0; i < clicksPerCode; i++ {
			wg.Add(1)
			c := code
			go func() {
				defer wg.Done()
				batcher.Record(c)
			}()
		}
	}
	wg.Wait()
	batcher.Stop()

	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, code := range codes {
		assert.Equal(t, int64(clicksPerCode), repo.perCode[code],
			"code %s: expected %d clicks", code, clicksPerCode)
	}
}

// TestClickBatcher_FlushInterval verifies that clicks are flushed
// automatically by the ticker without calling Stop.
func TestClickBatcher_FlushInterval(t *testing.T) {
	repo := newMockRepo()
	b := NewClickBatcher(repo, &mockLogger{})

	const n = 500
	for i := 0; i < n; i++ {
		b.Record("ticker_code")
	}

	// Wait for at least one tick (1 s + buffer).
	time.Sleep(1200 * time.Millisecond)

	repo.mu.Lock()
	got := repo.perCode["ticker_code"]
	repo.mu.Unlock()

	assert.Equal(t, int64(n), got, "ticker flush must have sent all %d clicks", n)
	b.Stop()
}

// TestClickBatcher_ChannelFullDrops verifies the batcher never blocks
// the caller when the channel is full — it simply drops the overflow.
func TestClickBatcher_ChannelFullDrops(t *testing.T) {
	repo := newMockRepo()
	// Use a tiny channel so it fills up instantly.
	b := &ClickBatcher{
		ch:     make(chan string, 10),
		repo:   repo,
		logger: &mockLogger{},
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	go b.run()

	// Send far more than the channel can hold — must not block.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10_000; i++ {
			b.Record("overflow")
		}
		close(done)
	}()

	select {
	case <-done:
		// good — returned without blocking
	case <-time.After(2 * time.Second):
		t.Fatal("Record() blocked when channel was full")
	}
	b.Stop()
}

// TestClickBatcher_StopFlushesRemaining ensures Stop() drains items
// that are still sitting in the channel buffer.
func TestClickBatcher_StopFlushesRemaining(t *testing.T) {
	repo := newMockRepo()
	b := NewClickBatcher(repo, &mockLogger{})

	const n = 200
	for i := 0; i < n; i++ {
		b.Record("stopdrain")
	}

	// Stop blocks until the worker has fully drained and flushed.
	b.Stop()

	repo.mu.Lock()
	got := repo.perCode["stopdrain"]
	repo.mu.Unlock()

	assert.Equal(t, int64(n), got,
		"Stop() must flush all %d buffered clicks", n)
}
