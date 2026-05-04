package handler

import (
	"io"
	"log/slog"
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
	ID               string `json:"id"`
	URL              string `json:"url"`
	ModerationStatus string `json:"moderationStatus"`
}

// Upload godoc
// @Summary Upload a photo
// @Description Uploads an image and returns its URL and moderation status.
// @Description The photo starts as "pending" and is approved/rejected asynchronously by Rekognition.
// @Description Only animal images are accepted. Use the returned URL in the report's photos field.
// @Tags uploads
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image file (jpeg, png or webp, max 5 MB)"
// @Success 201 {object} UploadResponse
// @Failure 400 {object} ErrorResponse "unsupported file type or malformed request"
// @Failure 401 {object} ErrorResponse "missing or invalid token"
// @Failure 413 {object} ErrorResponse "file exceeds maximum allowed size"
// @Failure 500 {object} ErrorResponse
// @Router /v1/uploads [post]
func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	maxBytes := h.service.MaxBytes()
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+512) // 512 extra for form overhead

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		slog.Warn("upload: parse multipart failed", "error", err)
		http.Error(w, "file too large or malformed request", http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		slog.Warn("upload: missing file field", "error", err)
		http.Error(w, "missing 'file' field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])

	slog.Info("upload: received file",
		"filename", header.Filename,
		"size_bytes", header.Size,
		"detected_content_type", contentType,
	)

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		slog.Error("upload: seek failed", "error", err)
		http.Error(w, "failed to process file", http.StatusInternalServerError)
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	meta, err := h.service.Upload(r.Context(), userID, file, header.Size, contentType)
	if err != nil {
		switch err {
		case uploaduc.ErrFileTooLarge:
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		case uploaduc.ErrUnsupportedType:
			slog.Warn("upload: unsupported type", "content_type", contentType, "filename", header.Filename)
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			slog.Error("upload: service error", "error", err, "user_id", userID, "filename", header.Filename)
			http.Error(w, "upload failed", http.StatusInternalServerError)
		}
		return
	}

	slog.Info("upload: success", "upload_id", meta.ID, "url", meta.URL, "user_id", userID)
	respondJSON(w, http.StatusCreated, UploadResponse{
		ID:               meta.ID,
		URL:              meta.URL,
		ModerationStatus: string(meta.ModerationStatus),
	})
}
