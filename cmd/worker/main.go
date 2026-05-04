package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"uptime-monitor/internal/logger"
	"uptime-monitor/internal/models"
	"uptime-monitor/internal/storage"
)

func main() {
	logger.InitLogger()
	defer logger.Log.Sync()

	dsn := "postgres://postgres:pass@localhost:5432/uptime?sslmode=disable"
	db := storage.InitDB(dsn)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.Log.Info("Воркер запущен. Начинаем мониторинг")
	pingTicker := time.NewTicker(5 * time.Second)
	defer pingTicker.Stop()
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer cleanupTicker.Stop()
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	go cleanupOldData(db, 7)
	for {
		select {
		case <-ctx.Done(): 
			logger.Log.Info("Получен сигнал завершения")
			sqlDB, _ := db.DB()
			sqlDB.Close()
			logger.Log.Info("Воркер успешно остановлен")
			return
		case <-pingTicker.C: 
			checkAllSites(db, client)
		case <-cleanupTicker.C:
			cleanupOldData(db, 7)
		}
	}
}


func checkAllSites(db *gorm.DB, client *http.Client) {
	logger.Log.Info("Запуск цикла проверки")
	var sites []models.Website
	err := db.Where("is_active = ? AND (last_checked_at IS NULL OR last_checked_at + (interval_seconds || ' seconds')::interval <= NOW())", true).Find(&sites).Error
	if err != nil {
		logger.Log.Error("Ошибка получения сайтов из БД", zap.Error(err))
		return
	}
	if len(sites) == 0 {
		logger.Log.Info("Нет активных сайтов для проверки")
		return
	}

	var wg sync.WaitGroup
	for _, site := range sites {
		wg.Add(1)
		go func(s models.Website) {
			defer wg.Done()
			start := time.Now()
			resp, err := client.Get(s.URL)
			duration := time.Since(start).Milliseconds()
			statusCode := 0
			if err == nil {
				statusCode = resp.StatusCode
				resp.Body.Close()
			}

			result := models.PingResult{
				WebsiteID:      s.ID,
				StatusCode:     statusCode,
				ResponseTimeMs: duration,
				CheckedAt:      time.Now(),
			}

			if err := db.Create(&result).Error; err != nil {
				logger.Log.Error("Ошибка записи результата", zap.Error(err))
			}
			if err := db.Model(&s).Update("last_checked_at", time.Now()).Error; err != nil {
				logger.Log.Error("Ошибка обновления последней даты проверки", zap.Error(err))
			}
		}(site)
	}
	wg.Wait()
	logger.Log.Info("Все проверки завершены")
}


func cleanupOldData(db *gorm.DB, daysToKeep int) {
	logger.Log.Info("Начинается очистка старых данных")
	cutoffTime := time.Now().AddDate(0, 0, -daysToKeep)
	result := db.Where("checked_at < ?", cutoffTime).Delete(&models.PingResult{})
	if result.Error != nil {
		logger.Log.Error("Ошибка при очистке старых данных", zap.Error(result.Error))
		return
	}
	logger.Log.Info("Очистка завершена", zap.Int64("удалено_строк", result.RowsAffected))
}