package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gattolab/wrappp/config"
	"github.com/gattolab/wrappp/pkg/common/exception"
	driver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

type DB struct {
	*gorm.DB
}

func buildDSN(host, port, user, password, database string) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, database, port,
	)
}

func NewDB(conf config.DatabaseConfig) (*DB, error) {

	maxPoolOpen := conf.MaxPoolOpen
	maxPoolIdle := conf.MaxPoolIdle
	maxPollLifeTime := conf.MaxPollLifeTime

	// Improved Log Level Handling
	var logLevel logger.LogLevel
	switch conf.LogLevel {
	case "INFO":
		logLevel = logger.Info
	case "WARN":
		logLevel = logger.Warn
	case "ERROR":
		logLevel = logger.Error
	default:
		logLevel = logger.Silent
	}

	loggerDB := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Primary (write) DSN — PgBouncer port 5432 → write pool
	writeDSN := buildDSN(conf.Host, conf.Port, conf.User, conf.Password, conf.Database)

	db, err := gorm.Open(driver.Open(writeDSN), &gorm.Config{
		Logger: loggerDB,
	})
	exception.PanicLogging(err)

	// Configure the primary connection pool
	sqlDB, err := db.DB()
	exception.PanicLogging(err)

	sqlDB.SetMaxOpenConns(maxPoolOpen)
	sqlDB.SetMaxIdleConns(maxPoolIdle)
	sqlDB.SetConnMaxLifetime(time.Duration(maxPollLifeTime) * time.Millisecond)

	if !conf.Standalone {
		// Read replica DSN — PgBouncer port 5433 → read pool
		// Falls back to primary host when DATABASE_READ_HOST is not set
		readHost := conf.ReadHost
		if readHost == "" {
			readHost = conf.Host
		}
		readDSN := buildDSN(readHost, conf.ReadPort, conf.User, conf.Password, conf.Database)

		err = db.Use(
			dbresolver.Register(dbresolver.Config{
				// Sources: write connections (same as primary)
				Sources: []gorm.Dialector{driver.Open(writeDSN)},
				// Replicas: read-only connections
				Replicas: []gorm.Dialector{driver.Open(readDSN)},
				// Random policy across replicas
				Policy: dbresolver.RandomPolicy{},
			}).
				SetMaxOpenConns(maxPoolOpen).
				SetMaxIdleConns(maxPoolIdle).
				SetConnMaxLifetime(time.Duration(maxPollLifeTime) * time.Millisecond),
		)
		exception.PanicLogging(err)
	}

	// Auto-migrate models (Uncomment when models are ready)
	// db.AutoMigrate(&entity.User{})

	return &DB{db}, nil
}
