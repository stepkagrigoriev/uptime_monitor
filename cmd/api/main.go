package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"uptime-monitor/internal/logger"
	"uptime-monitor/internal/models"
	"uptime-monitor/internal/storage"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	logger.InitLogger()
	defer logger.Log.Sync()

	dsn := "postgres://user:pass@localhost:5432/uptime?sslmode=disable"
	db = storage.InitDB(dsn)
	r := mux.NewRouter()

	r.Use(loggingMiddleware)
	r.HandleFunc("/sites", addSite).Methods(http.MethodPost)
	r.HandleFunc("/sites", getSites).Methods(http.MethodGet)
	r.HandleFunc("/sites/{id:[0-9]+}/stats", getSiteStats).Methods(http.MethodGet)
	r.HandleFunc("/sites/{id:[0-9]+}", deleteSite).Methods(http.MethodDelete)
	r.HandleFunc("/sites/{id:[0-9]+}/status", updateSiteStatus).Methods(http.MethodPatch)
	r.HandleFunc("/sites/{id:[0-9]+}/analytics", getSiteAnalytics).Methods(http.MethodGet)
	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	go func() {
		logger.Log.Info("API сервис запущен на порту :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Ошибка запуска сервера", zap.Error(err))
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logger.Log.Info("Получен сигнал остановки")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Ошибка при остановке сервера", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
	logger.Log.Info("API остановлен")
}


func addSite(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL             string `json:"url"`
		IntervalSeconds int    `json:"interval_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if input.IntervalSeconds < 1 {
		input.IntervalSeconds = 60
	}
	site := models.Website{
		URL:             input.URL,
		IntervalSeconds: input.IntervalSeconds,
	}
	if err := db.Create(&site).Error; err != nil {
		logger.Log.Error("Ошибка создания сайта", zap.Error(err))
		http.Error(w, "Ошибка БД", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(site)
}


func getSites(w http.ResponseWriter, r *http.Request) {
	var sites[]models.Website
	db.Find(&sites)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sites)
}


func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Log.Info("Входящий запрос", 
			zap.String("method", r.Method), 
			zap.String("url", r.URL.String()),
		)
		next.ServeHTTP(w, r)
	})
}


func getSiteStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	siteID := vars["id"]

	var stats []models.PingResult
	if err := db.Where("website_id = ?", siteID).Order("checked_at DESC").Limit(50).Find(&stats).Error; err != nil {
		http.Error(w, "Ошибка получения статистики", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}


func deleteSite(w http.ResponseWriter, r *http.Request) {
	siteID := mux.Vars(r)["id"]
	db.Where("website_id = ?", siteID).Delete(&models.PingResult{})
	if err := db.Delete(&models.Website{}, siteID).Error; err != nil {
		http.Error(w, "Ошибка при удалении", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent) 
}


func updateSiteStatus(w http.ResponseWriter, r *http.Request) {
	siteID := mux.Vars(r)["id"]

	var input struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Неверный JSON", http.StatusBadRequest)
		return
	}

	if err := db.Model(&models.Website{}).Where("id = ?", siteID).Update("is_active", input.IsActive).Error; err != nil {
		http.Error(w, "Ошибка обновления", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}


func getSiteAnalytics(w http.ResponseWriter, r *http.Request) {
	siteID := mux.Vars(r)["id"]

	var totalPings int64
	var successfulPings int64
	var avgResponseTime float64

	db.Model(&models.PingResult{}).Where("website_id = ?", siteID).Count(&totalPings)

	if totalPings == 0 {
		http.Error(w, "Нет данных для аналитики", http.StatusNotFound)
		return
	}

	db.Model(&models.PingResult{}).Where("website_id = ? AND status_code >= 200 AND status_code < 300", siteID).Count(&successfulPings)
	db.Model(&models.PingResult{}).Where("website_id = ?", siteID).Select("COALESCE(AVG(response_time_ms), 0)").Scan(&avgResponseTime)

	uptime := (float64(successfulPings) / float64(totalPings)) * 100
	response := map[string]interface{}{
		"website_id":           siteID,
		"total_checks":         totalPings,
		"uptime_percent":       uptime,
		"avg_response_time_ms": avgResponseTime,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}