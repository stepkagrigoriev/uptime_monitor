package main

import (
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"uptime-monitor/internal/logger"
	"uptime-monitor/internal/models"
	"uptime-monitor/internal/storage"
)

func main() {
	logger.InitLogger()
	defer logger.Log.Sync()

	dsn := "postgres://postgres:pass@localhost:5432/uptime?sslmode=disable"
	db := storage.InitDB(dsn)
	logger.Log.Info("Воркер запущен. Начинаем мониторинг...")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	for range ticker.C {
		logger.Log.Info("Запуск цикла проверки...")
		var sites []models.Website
		if err := db.Find(&sites).Error; err != nil {
			logger.Log.Error("Ошибка получения сайтов из БД", zap.Error(err))
			continue
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
				} else {
					logger.Log.Warn("Сайт недоступен", zap.String("url", s.URL), zap.Error(err))
				}
				result := models.PingResult{
					WebsiteID:      s.ID,
					StatusCode:     statusCode,
					ResponseTimeMs: duration,
					CheckedAt:      time.Now(),
				}

				if err := db.Create(&result).Error; err != nil {
					logger.Log.Error("Ошибка записи результата в БД", zap.Error(err))
				}
			}(site)
		}
		wg.Wait()
		logger.Log.Info("Цикл проверки завершен")
	}
}