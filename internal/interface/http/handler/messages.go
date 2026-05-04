package handler

import (
	"log/slog"
	"net/http"

	domain "github.com/yourname/go-backend/internal/domain/message"
	"github.com/yourname/go-backend/internal/interface/http/middleware"
	messageuc "github.com/yourname/go-backend/internal/usecase/message"
)

type MessageHandler struct {
	service *messageuc.Service
}

func NewMessageHandler(service *messageuc.Service) *MessageHandler {
	return &MessageHandler{service: service}
}

// StartConversation godoc
// @Summary Start (or resume) conversation about a report
// @Description Opens a conversation between the authenticated user and the report owner.
// @Description If a conversation for this report already exists for this user, the existing one is returned.
// @Tags messaging
// @Security BearerAuth
// @Produce json
// @Param id path string true "Report ID"
// @Success 200 {object} ConversationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /v1/reports/{id}/conversations [post]
func (h *MessageHandler) StartConversation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	reportID := r.PathValue("id")
	slog.Info("start conversation", "reportId", reportID, "requesterId", userID)
	out, err := h.service.StartConversation(r.Context(), reportID, userID)
	if err != nil {
		switch err {
		case messageuc.ErrReportNotFound:
			http.Error(w, "report not found", http.StatusNotFound)
		case messageuc.ErrSelfConversation:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	respondJSON(w, http.StatusOK, out)
}

// ListConversations godoc
// @Summary List current user conversations
// @Tags messaging
// @Security BearerAuth
// @Produce json
// @Success 200 {array} ConversationResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/conversations [get]
func (h *MessageHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	out, err := h.service.ListConversations(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, out)
}

// ListMessages godoc
// @Summary List messages from a conversation
// @Tags messaging
// @Security BearerAuth
// @Produce json
// @Param id path string true "Conversation ID"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 50, max 200)"
// @Success 200 {object} PagedMessagesResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /v1/conversations/{id}/messages [get]
func (h *MessageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	conversationID := r.PathValue("id")
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 50)
	out, err := h.service.ListMessages(r.Context(), conversationID, userID, page, pageSize)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	respondJSON(w, http.StatusOK, PagedMessagesResponse{Page: page, PageSize: pageSize, Items: toMessageResponses(out)})
}

// SendMessage godoc
// @Summary Send message to conversation
// @Tags messaging
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Param payload body SendMessageRequest true "Message payload"
// @Success 201 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /v1/conversations/{id}/messages [post]
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	conversationID := r.PathValue("id")
	var req SendMessageRequest
	if err := decodeJSON(r, &req); err != nil || req.Body == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	out, err := h.service.SendMessage(r.Context(), conversationID, userID, req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	respondJSON(w, http.StatusCreated, out)
}

func toMessageResponses(msgs []domain.Message) []MessageResponse {
	out := make([]MessageResponse, len(msgs))
	for i, m := range msgs {
		out[i] = MessageResponse{
			ID:             m.ID,
			ConversationID: m.ConversationID,
			SenderID:       m.SenderID,
			Body:           m.Body,
			CreatedAt:      m.CreatedAt.String(),
		}
	}
	return out
}
