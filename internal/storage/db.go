package storage

import (
	"uptime-monitor/internal/logger"
	"uptime-monitor/internal/models"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Log.Fatal("Не удалось подключиться к БД", zap.Error(err))
	}
	err = db.AutoMigrate(&models.Website{}, &models.PingResult{})
	if err != nil {
		logger.Log.Fatal("Ошибка автомиграции", zap.Error(err))
	}

	logger.Log.Info("Подключение к БД успешно, таблицы синхронизированы")
	return db
}