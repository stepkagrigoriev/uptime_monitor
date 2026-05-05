package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"uptime-monitor/internal/logger"
	"uptime-monitor/internal/storage"
	"uptime-monitor/internal/worker"
	"uptime-monitor/internal/config"
)

func main() {
	logger.InitLogger()
	defer logger.Log.Sync()
	cfg := config.LoadConfig()
	db := storage.InitDB(cfg.DBURL)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pingTicker := time.NewTicker(5 * time.Second)
	defer pingTicker.Stop()
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer cleanupTicker.Stop()

	client := &http.Client{Timeout: 5 * time.Second}
	go worker.CleanupOldData(db, 7)
	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Завершение работы воркера...")
			sqlDB, _ := db.DB()
			sqlDB.Close()
			return
		case <-pingTicker.C:
			worker.CheckAllSites(db, client)
		case <-cleanupTicker.C:
			go worker.CleanupOldData(db, 7)
		}
	}
}