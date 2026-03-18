package domain

import (
	"context"
	"time"

	"github.com/gattolab/wrappp/internal/domain/entity"
)

type ShortUrlService interface {
	Create(ctx context.Context, payload CreateShortUrlPayload) (ShortUrl, error)
	GetByCode(ctx context.Context, code string, isRedirect bool) (*ShortUrl, error)
	GetAll(ctx context.Context) ([]ShortUrl, error)
	DeleteByCode(ctx context.Context, code string) error
}

type ShortUrlRepository interface {
	Create(ctx context.Context, shortUrl entity.ShortUrl) (entity.ShortUrl, error)
	Update(ctx context.Context, shortUrl *entity.ShortUrl) (*entity.ShortUrl, error)
	GetByCode(ctx context.Context, code string) (*entity.ShortUrl, error)
	GetAll(ctx context.Context) ([]entity.ShortUrl, error)
	DeleteByCode(ctx context.Context, code string) error
	IncrementClick(ctx context.Context, code string) error
	IncrementClickBy(ctx context.Context, code string, n int64) error
}

type CreateShortUrlPayload struct {
	TargetUrl string     `json:"target_url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type ShortUrl struct {
	ID             int64      `json:"id"`
	ShortCode      string     `json:"code"`
	OriginalUrl    string     `json:"target_url"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	LastAccessedAt *time.Time `json:"last_accessed_at"`
	IsActive       bool       `json:"is_active"`
	ClickCount     int64      `json:"click_count"`
}
