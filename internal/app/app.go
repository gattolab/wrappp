package app

import (
	"github.com/gattolab/wrappp/config"
	"github.com/gattolab/wrappp/pkg/cache"
	"github.com/gattolab/wrappp/pkg/db"
	"github.com/gattolab/wrappp/pkg/logger"
	"github.com/gofiber/fiber/v2"

	shorturlhandler "github.com/gattolab/wrappp/internal/usecase/shorturl/controller/http"
	shorturlrepository "github.com/gattolab/wrappp/internal/usecase/shorturl/repository"
	shorturlservice "github.com/gattolab/wrappp/internal/usecase/shorturl/service"
)

func NewApplication(api fiber.Router, logger logger.Logger, db *db.DB, cache cache.Engine, config *config.Configuration) {
	v1 := api.Group("/v1")

	shortUrlRepository := shorturlrepository.NewShortUrlRepository(db, logger, cache, config)
	shortUrlService := shorturlservice.NewShortUrlService(shortUrlRepository, cache, logger, config)
	shortUrlHandler := shorturlhandler.NewShortUrlHandler(shortUrlService, config)
	shortUrlHandler.InitRoute(v1)

}
