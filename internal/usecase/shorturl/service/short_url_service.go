package service

import (
	"context"
	"errors"
	"time"

	"github.com/gattolab/wrappp/config"
	"github.com/gattolab/wrappp/internal/domain"
	"github.com/gattolab/wrappp/internal/domain/entity"
	"github.com/gattolab/wrappp/pkg/cache"
	"github.com/gattolab/wrappp/pkg/logger"
	"github.com/gattolab/wrappp/pkg/utils"
	"github.com/segmentio/ksuid"
)

type ShortUrlService struct {
	ShortUrlRepository domain.ShortUrlRepository
	Cache              cache.Engine
	Logger             logger.Logger
	Conf               *config.Configuration
	ClickBatcher       *ClickBatcher
}

func NewShortUrlService(shortUrlRepository domain.ShortUrlRepository, cache cache.Engine, logger logger.Logger, conf *config.Configuration) domain.ShortUrlService {
	return &ShortUrlService{
		ShortUrlRepository: shortUrlRepository,
		Cache:              cache,
		Logger:             logger,
		Conf:               conf,
		ClickBatcher:       NewClickBatcher(shortUrlRepository, logger),
	}
}

func (s *ShortUrlService) Create(ctx context.Context, payload domain.CreateShortUrlPayload) (domain.ShortUrl, error) {
	code := ksuid.New().String()[:8]
	now := time.Now()
	newShortUrl := entity.ShortUrl{
		ShortCode:   code,
		OriginalUrl: payload.TargetUrl,
		CreatedAt:   now,
		IsActive:    true,
		ExpiresAt:   payload.ExpiresAt,
	}

	created, err := s.ShortUrlRepository.Create(ctx, newShortUrl)
	if err != nil {
		return domain.ShortUrl{}, err
	}

	return domain.ShortUrl{
		ID:             created.ID,
		ShortCode:      created.ShortCode,
		OriginalUrl:    created.OriginalUrl,
		CreatedAt:      created.CreatedAt,
		ExpiresAt:      created.ExpiresAt,
		LastAccessedAt: created.LastAccessedAt,
		IsActive:       created.IsActive,
		ClickCount:     created.ClickCount,
	}, nil
}

func (s *ShortUrlService) GetByCode(ctx context.Context, code string, isRedirect bool) (*domain.ShortUrl, error) {
	data, err := utils.UseCache[*entity.ShortUrl](
		ctx,
		s.Cache,
		"short_url:"+code,
		func(ctx context.Context) (*entity.ShortUrl, error) {
			return s.ShortUrlRepository.GetByCode(ctx, code)
		},
		10*time.Second,
	)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, errors.New("not found")
	}

	if isRedirect {
		s.ClickBatcher.Record(code)
	}

	return &domain.ShortUrl{
		ID:             data.ID,
		ShortCode:      data.ShortCode,
		OriginalUrl:    data.OriginalUrl,
		CreatedAt:      data.CreatedAt,
		ExpiresAt:      data.ExpiresAt,
		LastAccessedAt: data.LastAccessedAt,
		IsActive:       data.IsActive,
		ClickCount:     data.ClickCount,
	}, nil
}

func (s *ShortUrlService) GetAll(ctx context.Context) ([]domain.ShortUrl, error) {
	list, err := s.ShortUrlRepository.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]domain.ShortUrl, 0, len(list))
	for _, item := range list {
		results = append(results, domain.ShortUrl{
			ID:             item.ID,
			ShortCode:      item.ShortCode,
			OriginalUrl:    item.OriginalUrl,
			CreatedAt:      item.CreatedAt,
			ExpiresAt:      item.ExpiresAt,
			LastAccessedAt: item.LastAccessedAt,
			IsActive:       item.IsActive,
			ClickCount:     item.ClickCount,
		})
	}

	return results, nil
}

func (s *ShortUrlService) DeleteByCode(ctx context.Context, code string) error {
	return s.ShortUrlRepository.DeleteByCode(ctx, code)
}
