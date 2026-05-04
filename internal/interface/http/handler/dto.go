package handler

import "time"

type ErrorResponse struct {
	Message string `json:"message" example:"invalid payload"`
}

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

type UserResponse struct {
	ID        string `json:"id"                  example:"64a1f2c3d4e5f6789abcdef0"`
	Name      string `json:"name"                example:"João Silva"`
	Email     string `json:"email"               example:"joao@example.com"`
	AvatarURL string `json:"avatarUrl,omitempty" example:"https://cdn.example.com/avatar.jpg"`
	City      string `json:"city,omitempty"      example:"São Paulo"`
	State     string `json:"state,omitempty"     example:"SP"`
}

type LocationResponse struct {
	Latitude  float64 `json:"latitude"  example:"-23.5505"`
	Longitude float64 `json:"longitude" example:"-46.6333"`
}

// ReportResponse is the full representation of a found-animal report.
// Photos contains the public URLs of the uploaded images.
// The report is only listed publicly once all photos pass moderation (status "approved").
type ReportResponse struct {
	ID                    string           `json:"id"                    example:"64a1f2c3d4e5f6789abcdef0"`
	OwnerID               string           `json:"ownerId"               example:"64a1f2c3d4e5f6789abcdef1"`
	PetType               string           `json:"petType"               example:"dog"`
	Title                 string           `json:"title"                 example:"Golden Retriever found in Pinheiros"`
	Description           string           `json:"description"           example:"Friendly dog, no collar, found near the park"`
	Characteristics       []string         `json:"characteristics"       example:"golden,male,large"`
	LastSeenLocation      LocationResponse `json:"lastSeenLocation"`
	Photos                []string         `json:"photos"                example:"https://bucket.s3.amazonaws.com/abc.jpg"`
	IsShelteredByReporter bool             `json:"isShelteredByReporter" example:"true"`
	Status                string           `json:"status"                example:"open"`
	CreatedAt             time.Time        `json:"createdAt"`
	UpdatedAt             time.Time        `json:"updatedAt"`
}

type StartOAuthResponse struct {
	AuthURL string `json:"authUrl" example:"https://accounts.google.com/o/oauth2/auth?..."`
	State   string `json:"state"   example:"uuid.hmac"`
}

type OAuthCallbackResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"accessToken"  example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string       `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type RegisterRequest struct {
	Name     string `json:"name"     example:"João Silva"`
	Email    string `json:"email"    example:"joao@example.com"`
	Password string `json:"password" example:"s3cr3tP@ss"`
}

type LoginRequest struct {
	Email    string `json:"email"    example:"joao@example.com"`
	Password string `json:"password" example:"s3cr3tP@ss"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type RefreshResponse struct {
	AccessToken  string `json:"accessToken"  example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refreshToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type UpdateMeRequest struct {
	Name  string `json:"name"  example:"João Silva"`
	City  string `json:"city"  example:"São Paulo"`
	State string `json:"state" example:"SP"`
}

// CreateReportRequest creates a new found-animal report.
// Upload photos first via POST /v1/uploads and include the returned URLs in the photos field.
// The report becomes publicly visible once all photos pass moderation.
type CreateReportRequest struct {
	PetType               string   `json:"petType"               example:"dog"`
	Title                 string   `json:"title"                 example:"Golden Retriever found in Pinheiros"`
	Description           string   `json:"description"           example:"Friendly dog, no collar, found near the park"`
	Characteristics       []string `json:"characteristics"       example:"golden,male,large"`
	Latitude              float64  `json:"latitude"              example:"-23.5505"`
	Longitude             float64  `json:"longitude"             example:"-46.6333"`
	Photos                []string `json:"photos"                example:"https://bucket.s3.amazonaws.com/abc.jpg"`
	IsShelteredByReporter bool     `json:"isShelteredByReporter" example:"true"`
}

// PatchReportRequest updates an existing report. Only the owner can patch.
// To replace photos, include all desired URLs (old + new) in the photos field.
type PatchReportRequest struct {
	Title                 string   `json:"title"                 example:"Golden Retriever found in Pinheiros"`
	Description           string   `json:"description"           example:"Friendly dog, no collar, found near the park"`
	Characteristics       []string `json:"characteristics"       example:"golden,male,large"`
	Latitude              float64  `json:"latitude"              example:"-23.5505"`
	Longitude             float64  `json:"longitude"             example:"-46.6333"`
	Photos                []string `json:"photos"                example:"https://bucket.s3.amazonaws.com/abc.jpg"`
	IsShelteredByReporter bool     `json:"isShelteredByReporter" example:"true"`
}

type SendMessageRequest struct {
	Body string `json:"body" example:"Is the dog still with you? I think it's mine."`
}

type ConversationResponse struct {
	ID            string   `json:"id"            example:"64a1f2c3d4e5f6789abcdef2"`
	ReportID      string   `json:"reportId"      example:"64a1f2c3d4e5f6789abcdef0"`
	Participants  []string `json:"participants"  example:"64a1f2c3d4e5f6789abcdef1,64a1f2c3d4e5f6789abcdef3"`
	LastMessageAt string   `json:"lastMessageAt" example:"2024-01-15T10:30:00Z"`
}

type MessageResponse struct {
	ID             string `json:"id"             example:"64a1f2c3d4e5f6789abcdef4"`
	ConversationID string `json:"conversationId" example:"64a1f2c3d4e5f6789abcdef2"`
	SenderID       string `json:"senderId"       example:"64a1f2c3d4e5f6789abcdef1"`
	Body           string `json:"body"           example:"Is the dog still with you? I think it's mine."`
	CreatedAt      string `json:"createdAt"      example:"2024-01-15T10:30:00Z"`
}

type PagedReportsResponse struct {
	Page     int              `json:"page"     example:"1"`
	PageSize int              `json:"pageSize" example:"20"`
	Items    []ReportResponse `json:"items"`
}

type PagedMessagesResponse struct {
	Page     int               `json:"page"     example:"1"`
	PageSize int               `json:"pageSize" example:"50"`
	Items    []MessageResponse `json:"items"`
}

type ConversationsResponse = []ConversationResponse
type MessagesResponse = []MessageResponse
