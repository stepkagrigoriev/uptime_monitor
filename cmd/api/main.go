package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"uptime-monitor/internal/api/handlers"
	"uptime-monitor/internal/api/middleware"
	"uptime-monitor/internal/logger"
	"uptime-monitor/internal/storage"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func main() {
	logger.InitLogger()
	defer logger.Log.Sync()

	db := storage.InitDB("postgres://postgres:pass@localhost:5432/uptime?sslmode=disable")
	h := handlers.NewSiteHandler(db)

	r := mux.NewRouter()
	r.Use(middleware.Logging)

	r.HandleFunc("/sites", h.AddSite).Methods(http.MethodPost)
	r.HandleFunc("/sites", h.GetSites).Methods(http.MethodGet)
	r.HandleFunc("/sites/{id:[0-9]+}/stats", h.GetSiteStats).Methods(http.MethodGet)
	r.HandleFunc("/sites/{id:[0-9]+}", h.DeleteSite).Methods(http.MethodDelete)
	r.HandleFunc("/sites/{id:[0-9]+}/status", h.UpdateSiteStatus).Methods(http.MethodPatch)
	r.HandleFunc("/sites/{id:[0-9]+}/analytics", h.GetSiteAnalytics).Methods(http.MethodGet)

	server := &http.Server{Addr: ":8080", Handler: r}

	go func() {
		logger.Log.Info("API сервис запущен на :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Ошибка сервера", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	sqlDB, _ := db.DB()
	sqlDB.Close()
	logger.Log.Info("API остановлен")
}