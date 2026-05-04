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

func (h *SiteHandler) AddSite(w http.ResponseWriter, r *http.Request) {
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

	site := models.Website{URL: input.URL, IntervalSeconds: input.IntervalSeconds}
	if err := h.DB.Create(&site).Error; err != nil {
		logger.Log.Error("Ошибка БД", zap.Error(err))
		http.Error(w, "Ошибка БД", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(site)
}

func (h *SiteHandler) GetSites(w http.ResponseWriter, r *http.Request) {
	var sites[]models.Website
	h.DB.Find(&sites)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sites)
}

func (h *SiteHandler) GetSiteStats(w http.ResponseWriter, r *http.Request) {
	siteID := mux.Vars(r)["id"]
	var stats[]models.PingResult

	h.DB.Where("website_id = ?", siteID).Order("checked_at DESC").Limit(50).Find(&stats)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *SiteHandler) DeleteSite(w http.ResponseWriter, r *http.Request) {
	siteID := mux.Vars(r)["id"]
	h.DB.Where("website_id = ?", siteID).Delete(&models.PingResult{})
	h.DB.Delete(&models.Website{}, siteID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *SiteHandler) UpdateSiteStatus(w http.ResponseWriter, r *http.Request) {
	siteID := mux.Vars(r)["id"]
	var input struct {
		IsActive bool `json:"is_active"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	h.DB.Model(&models.Website{}).Where("id = ?", siteID).Update("is_active", input.IsActive)
	w.WriteHeader(http.StatusOK)
}

func (h *SiteHandler) GetSiteAnalytics(w http.ResponseWriter, r *http.Request) {
	siteID := mux.Vars(r)["id"]
	var totalPings, successfulPings int64
	var avgResponseTime float64

	h.DB.Model(&models.PingResult{}).Where("website_id = ?", siteID).Count(&totalPings)
	if totalPings == 0 {
		http.Error(w, "Нет данных", http.StatusNotFound)
		return
	}

	h.DB.Model(&models.PingResult{}).Where("website_id = ? AND status_code >= 200 AND status_code < 300", siteID).Count(&successfulPings)
	h.DB.Model(&models.PingResult{}).Where("website_id = ?", siteID).Select("COALESCE(AVG(response_time_ms), 0)").Scan(&avgResponseTime)

	uptime := (float64(successfulPings) / float64(totalPings)) * 100

	json.NewEncoder(w).Encode(map[string]interface{}{
		"website_id":           siteID,
		"total_checks":         totalPings,
		"uptime_percent":       uptime,
		"avg_response_time_ms": avgResponseTime,
	})
}