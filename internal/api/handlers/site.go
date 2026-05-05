package handlers

import (
	"encoding/json"
	"net/http"
	"uptime-monitor/internal/logger"
	"uptime-monitor/internal/models"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SiteHandler struct {
	DB *gorm.DB
}

func NewSiteHandler(db *gorm.DB) *SiteHandler {
	return &SiteHandler{DB: db}
}

func (h *SiteHandler) getUserID(r *http.Request) uint {
	return r.Context().Value("user_id").(uint)
}

func (h *SiteHandler) AddSite(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)

	var input struct {
		URL             string `json:"url"`
		IntervalSeconds int    `json:"interval_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Неверный формат данных", http.StatusBadRequest)
		return
	}

	if input.IntervalSeconds < 1 {
		input.IntervalSeconds = 60
	}

	site := models.Website{
		URL:             input.URL,
		IntervalSeconds: input.IntervalSeconds,
		UserID:          userID,
	}
	if err := h.DB.Create(&site).Error; err != nil {
		logger.Log.Error("Ошибка при сохранении сайта в БД", zap.Error(err))
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(site)
}

func (h *SiteHandler) GetSites(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	var sites []models.Website

	h.DB.Where("user_id = ?", userID).Find(&sites)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sites)
}

func (h *SiteHandler) GetSiteStats(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	siteID := mux.Vars(r)["id"]
	var site models.Website
	if err := h.DB.Where("id = ? AND user_id = ?", siteID, userID).First(&site).Error; err != nil {
		http.Error(w, "Сайт не найден или доступ запрещен", http.StatusNotFound)
		return
	}
	var stats []models.PingResult
	h.DB.Where("website_id = ?", siteID).Order("checked_at DESC").Limit(50).Find(&stats)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *SiteHandler) DeleteSite(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	siteID := mux.Vars(r)["id"]
	result := h.DB.Where("id = ? AND user_id = ?", siteID, userID).First(&models.Website{})
	if result.Error != nil {
		http.Error(w, "Сайт не найден или доступ запрещен", http.StatusNotFound)
		return
	}
	h.DB.Where("website_id = ?", siteID).Delete(&models.PingResult{})
	h.DB.Delete(&models.Website{}, siteID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *SiteHandler) UpdateSiteStatus(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	siteID := mux.Vars(r)["id"]
	var input struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}
	result := h.DB.Model(&models.Website{}).Where("id = ? AND user_id = ?", siteID, userID).Update("is_active", input.IsActive)
	if result.Error != nil || result.RowsAffected == 0 {
		http.Error(w, "Сайт не найден или доступ запрещен", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *SiteHandler) GetSiteAnalytics(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	siteID := mux.Vars(r)["id"]
	var site models.Website
	if err := h.DB.Where("id = ? AND user_id = ?", siteID, userID).First(&site).Error; err != nil {
		http.Error(w, "Сайт не найден или доступ запрещен", http.StatusNotFound)
		return
	}

	var totalPings, successfulPings int64
	var avgResponseTime float64

	h.DB.Model(&models.PingResult{}).Where("website_id = ?", siteID).Count(&totalPings)
	if totalPings == 0 {
		http.Error(w, "Нет данных для аналитики", http.StatusNotFound)
		return
	}

	h.DB.Model(&models.PingResult{}).Where("website_id = ? AND status_code >= 200 AND status_code < 300", siteID).Count(&successfulPings)
	h.DB.Model(&models.PingResult{}).Where("website_id = ?", siteID).Select("COALESCE(AVG(response_time_ms), 0)").Scan(&avgResponseTime)

	uptime := (float64(successfulPings) / float64(totalPings)) * 100

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"website_id":           siteID,
		"url":                  site.URL,
		"total_checks":         totalPings,
		"uptime_percent":       uptime,
		"avg_response_time_ms": avgResponseTime,
	})
}