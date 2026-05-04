package worker

import (
	"net/http"
	"sync"
	"time"

	"uptime-monitor/internal/logger"
	"uptime-monitor/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)


func CheckAllSites(db *gorm.DB, client *http.Client) {
	var targets []models.Website
	err := db.Where("is_active = ? AND (last_checked_at IS NULL OR last_checked_at + (interval_seconds || ' seconds')::interval <= NOW())", true).Find(&targets).Error
	if err != nil {
		logger.Log.Error("Ошибка получения сайтов", zap.Error(err))
		return
	}
	if len(targets) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, site := range targets {
		wg.Add(1)
		go func(site models.Website) {
			defer wg.Done()

			start := time.Now()
			resp, err := client.Get(site.URL)
			duration := time.Since(start).Milliseconds()

			statusCode := 0
			if err == nil {
				statusCode = resp.StatusCode
				resp.Body.Close()
			}

			result := models.PingResult{
				WebsiteID:      site.ID,
				StatusCode:     statusCode,
				ResponseTimeMs: duration,
				CheckedAt:      time.Now(),
			}
			if err := db.Create(&result).Error; err != nil {
				logger.Log.Error("Ошибка записи результата", zap.Error(err))
			}
			if err := db.Model(&site).Update("last_checked_at", time.Now()).Error; err != nil {
				logger.Log.Error("Ошибка обновления последней даты проверки", zap.Error(err))
			}
		}(site)
	}
	wg.Wait()
}

func CleanupOldData(db *gorm.DB, days int) {
	logger.Log.Info("Запуск очистки данных", zap.Int("days", days))
	cutoff := time.Now().AddDate(0, 0, -days)
	
	res := db.Where("checked_at < ?", cutoff).Delete(&models.PingResult{})
	if res.Error != nil {
		logger.Log.Error("Ошибка очистки", zap.Error(res.Error))
		return
	}
	logger.Log.Info("Очистка завершена", zap.Int64("deleted_rows", res.RowsAffected))
}