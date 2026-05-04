package handler

import "time"

type ErrorResponse struct {
	Message string `json:"message"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	City      string `json:"city,omitempty"`
	State     string `json:"state,omitempty"`
}

type LocationResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type ReportResponse struct {
	ID                    string           `json:"id"`
	OwnerID               string           `json:"ownerId"`
	PetType               string           `json:"petType"`
	Title                 string           `json:"title"`
	Description           string           `json:"description"`
	Characteristics       []string         `json:"characteristics"`
	LastSeenLocation      LocationResponse `json:"lastSeenLocation"`
	Photos                []string         `json:"photos"`
	IsShelteredByReporter bool             `json:"isShelteredByReporter"`
	Status                string           `json:"status"`
	CreatedAt             time.Time        `json:"createdAt"`
	UpdatedAt             time.Time        `json:"updatedAt"`
}

type StartOAuthResponse struct {
	AuthURL string `json:"authUrl"`
	State   string `json:"state"`
}

type OAuthCallbackResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type RefreshResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type UpdateMeRequest struct {
	Name  string `json:"name"`
	City  string `json:"city"`
	State string `json:"state"`
}

type CreateReportRequest struct {
	PetType               string   `json:"petType"`
	Title                 string   `json:"title"`
	Description           string   `json:"description"`
	Characteristics       []string `json:"characteristics"`
	Latitude              float64  `json:"latitude"`
	Longitude             float64  `json:"longitude"`
	Photos                []string `json:"photos"`
	IsShelteredByReporter bool     `json:"isShelteredByReporter"`
}

type PatchReportRequest struct {
	Title                 string   `json:"title"`
	Description           string   `json:"description"`
	Characteristics       []string `json:"characteristics"`
	Latitude              float64  `json:"latitude"`
	Longitude             float64  `json:"longitude"`
	Photos                []string `json:"photos"`
	IsShelteredByReporter bool     `json:"isShelteredByReporter"`
}

type SendMessageRequest struct {
	Body string `json:"body"`
}

type ConversationResponse struct {
	ID            string   `json:"id"`
	ReportID      string   `json:"reportId"`
	Participants  []string `json:"participants"`
	LastMessageAt string   `json:"lastMessageAt"`
}

type MessageResponse struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
	SenderID       string `json:"senderId"`
	Body           string `json:"body"`
	CreatedAt      string `json:"createdAt"`
}

type PagedReportsResponse struct {
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
	Items    []ReportResponse `json:"items"`
}

type PagedMessagesResponse struct {
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Items    []MessageResponse `json:"items"`
}

type ConversationsResponse = []ConversationResponse
type MessagesResponse = []MessageResponse
