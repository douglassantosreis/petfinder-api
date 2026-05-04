package handler

import (
	"io"
	"net/http"

	"github.com/yourname/go-backend/internal/interface/http/middleware"
	uploaduc "github.com/yourname/go-backend/internal/usecase/upload"
)

type UploadHandler struct {
	service *uploaduc.Service
}

func NewUploadHandler(service *uploaduc.Service) *UploadHandler {
	return &UploadHandler{service: service}
}

type UploadResponse struct {
	URL string `json:"url"`
}

// Upload godoc
// @Summary Upload a photo
// @Tags uploads
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image file (jpeg, png or webp, max 5 MB)"
// @Success 201 {object} UploadResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse
// @Router /v1/uploads [post]
func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	maxBytes := h.service.MaxBytes()
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+512) // 512 extra for form overhead

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		http.Error(w, "file too large or malformed request", http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing 'file' field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Detect content type from the actual bytes, not the Content-Type header.
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])

	// Rewind so the service reads the full file.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "failed to process file", http.StatusInternalServerError)
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	url, err := h.service.Upload(r.Context(), userID, file, header.Size, contentType)
	if err != nil {
		switch err {
		case uploaduc.ErrFileTooLarge:
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		case uploaduc.ErrUnsupportedType:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "upload failed", http.StatusInternalServerError)
		}
		return
	}

	respondJSON(w, http.StatusCreated, UploadResponse{URL: url})
}
