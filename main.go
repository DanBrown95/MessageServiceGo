package main

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"syscall"
	"time"

	iconfig "github.com/rerolldrinks/messageservice/config"
	"github.com/rerolldrinks/messageservice/models"
	"github.com/rerolldrinks/messageservice/services"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := iconfig.LoadConfig(); err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	env := os.Getenv("ENV")
	if env == "local" || env == "" {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Str("env", env).Logger()
	} else {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Str("env", env).Logger()
	}
	log.Info().Msg("MessageServiceGo starting")

	db, err := sql.Open("sqlserver", iconfig.AppConfig.SqlAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open SQL connection")
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping SQL Server")
	}
	log.Info().Msg("Connected to SQL Server")

	sqlService := services.NewSQLService(db)
	broadcastFunc := services.NewHTTPBroadcaster(iconfig.AppConfig.MonitorAPIAddress)
	webhookService := services.NewWebhookService(&iconfig.AppConfig.MessageSettings, broadcastFunc)

	interval := time.Duration(iconfig.AppConfig.ProcessingPollingIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info().Dur("interval", interval).Msg("Polling started")

	for {
		select {
		case <-ticker.C:
			processMessages(ctx, sqlService, webhookService)
		case <-ctx.Done():
			log.Info().Msg("Shutting down...")
			return
		}
	}
}

func processMessages(ctx context.Context, sqlSvc services.SQLService, webhookSvc services.WebhookService) {
	now := time.Now().UTC()

	messages, err := sqlSvc.GetActiveMessages(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query active messages")
		return
	}

	for _, msg := range messages {
		if !isDue(msg, now) {
			continue
		}

		log.Info().Int("id", msg.Id).Str("message", msg.Message).Msg("Broadcasting message")

		if err := webhookSvc.TriggerWebhook(ctx, msg); err != nil {
			log.Error().Err(err).Int("id", msg.Id).Msg("Failed to broadcast message")
			continue
		}

		if err := sqlSvc.UpdateLastRunUTC(ctx, msg.Id, now); err != nil {
			log.Error().Err(err).Int("id", msg.Id).Msg("Failed to update LastRunUTC")
		}
	}
}

// isDue returns true if the message has never been sent or its interval has elapsed.
func isDue(msg models.MessageRecord, now time.Time) bool {
	if msg.LastRunUTC == nil {
		return true
	}
	nextRun := msg.LastRunUTC.Add(time.Duration(msg.IntervalMinutes) * time.Minute)
	return now.After(nextRun)
}
