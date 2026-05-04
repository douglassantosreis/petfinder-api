package handler

import (
	"net/http"
	"strconv"

	domain "github.com/yourname/go-backend/internal/domain/ad"
	"github.com/yourname/go-backend/internal/interface/http/middleware"
	aduc "github.com/yourname/go-backend/internal/usecase/ad"
)

type ReportHandler struct {
	service *aduc.Service
}

func NewReportHandler(service *aduc.Service) *ReportHandler {
	return &ReportHandler{service: service}
}

// Create godoc
// @Summary Create found animal report
// @Tags reports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body CreateReportRequest true "Report payload"
// @Success 201 {object} ReportResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/reports [post]
func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req CreateReportRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	out, err := h.service.Create(r.Context(), aduc.CreateInput{
		OwnerID:               userID,
		PetType:               req.PetType,
		Title:                 req.Title,
		Description:           req.Description,
		Characteristics:       req.Characteristics,
		Latitude:              req.Latitude,
		Longitude:             req.Longitude,
		Photos:                req.Photos,
		IsShelteredByReporter: req.IsShelteredByReporter,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusCreated, toReportResponse(out))
}

// GetByID godoc
// @Summary Get report by ID
// @Tags reports
// @Security BearerAuth
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {object} ReportResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/reports/{id} [get]
func (h *ReportHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	out, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	respondJSON(w, http.StatusOK, toReportResponse(out))
}

// List godoc
// @Summary List open reports
// @Tags reports
// @Security BearerAuth
// @Produce json
// @Param page      query int    false "Page number (default 1)"
// @Param page_size query int    false "Items per page (default 20, max 100)"
// @Param lat       query number false "Latitude — when provided results are sorted by distance"
// @Param lng       query number false "Longitude — required when lat is set"
// @Param radius_km query number false "Search radius in km (default 0.5 = 500m)"
// @Success 200 {object} PagedReportsResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/reports [get]
func (h *ReportHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)

	var geo *aduc.GeoFilter
	if lat, lng, ok := queryGeo(r); ok {
		geo = &aduc.GeoFilter{
			Latitude:  lat,
			Longitude: lng,
			RadiusKm:  queryFloat(r, "radius_km", 0),
		}
	}

	out, err := h.service.ListOpen(r.Context(), page, pageSize, geo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	items := make([]ReportResponse, len(out))
	for i, r := range out {
		items[i] = toReportResponse(r)
	}
	respondJSON(w, http.StatusOK, PagedReportsResponse{Page: page, PageSize: pageSize, Items: items})
}

// Patch godoc
// @Summary Update found animal report
// @Tags reports
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Report ID"
// @Param payload body PatchReportRequest true "Report payload"
// @Success 200 {object} ReportResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /v1/reports/{id} [patch]
func (h *ReportHandler) Patch(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	reportID := r.PathValue("id")
	var req PatchReportRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	out, err := h.service.Update(r.Context(), userID, reportID, domain.FoundAnimalReport{
		Title:                 req.Title,
		Description:           req.Description,
		Characteristics:       req.Characteristics,
		LastSeenLocation:      domain.NewGeoPoint(req.Latitude, req.Longitude),
		Photos:                req.Photos,
		IsShelteredByReporter: req.IsShelteredByReporter,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	respondJSON(w, http.StatusOK, toReportResponse(out))
}

// Resolve godoc
// @Summary Mark report as resolved
// @Tags reports
// @Security BearerAuth
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {object} ReportResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /v1/reports/{id}/resolve [post]
func (h *ReportHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	reportID := r.PathValue("id")
	out, err := h.service.Resolve(r.Context(), userID, reportID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	respondJSON(w, http.StatusOK, toReportResponse(out))
}

// Archive godoc
// @Summary Archive report
// @Tags reports
// @Security BearerAuth
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {object} ReportResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /v1/reports/{id}/archive [post]
func (h *ReportHandler) Archive(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	reportID := r.PathValue("id")
	out, err := h.service.Archive(r.Context(), userID, reportID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	respondJSON(w, http.StatusOK, toReportResponse(out))
}

// --- helpers ---

func toReportResponse(r domain.FoundAnimalReport) ReportResponse {
	return ReportResponse{
		ID:      r.ID,
		OwnerID: r.OwnerID,
		PetType: r.PetType,
		Title:   r.Title,
		Description:           r.Description,
		Characteristics:       r.Characteristics,
		LastSeenLocation:      LocationResponse{Latitude: r.LastSeenLocation.Latitude(), Longitude: r.LastSeenLocation.Longitude()},
		Photos:                r.Photos,
		IsShelteredByReporter: r.IsShelteredByReporter,
		Status:                string(r.Status),
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
	}
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

func queryFloat(r *http.Request, key string, defaultVal float64) float64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return defaultVal
	}
	return f
}

// queryGeo returns lat, lng and true when both are present and non-zero.
func queryGeo(r *http.Request) (lat, lng float64, ok bool) {
	lat = queryFloat(r, "lat", 0)
	lng = queryFloat(r, "lng", 0)
	ok = lat != 0 || lng != 0
	return
}
