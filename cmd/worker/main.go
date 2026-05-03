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
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	for {
		select {
		case <-ctx.Done(): 
			logger.Log.Info("Получен сигнал завершения")
			sqlDB, _ := db.DB()
			sqlDB.Close()
			logger.Log.Info("Воркер успешно остановлен")
			return
		case <-ticker.C: 
			checkAllSites(db, client)
		}
	}
}


func checkAllSites(db *gorm.DB, client *http.Client) {
	logger.Log.Info("Запуск цикла проверки")
	var sites []models.Website
	if err := db.Where("is_active = ?", true).Find(&sites).Error; err != nil {
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
		}(site)
	}
	wg.Wait()
	logger.Log.Info("Все проверки завершены")
}