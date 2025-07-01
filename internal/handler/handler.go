package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/0xFEE1DEADatm/goAuthAPI/internal/middleware"
	"github.com/0xFEE1DEADatm/goAuthAPI/internal/token"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

type TokensResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type GetTokensRequest struct {
	UserGUID string `json:"user_guid"`
}

// GetTokens godoc
// @Summary Получение access и refresh токенов
// @Description Выдаёт пару токенов (access и refresh) по GUID пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param input body GetTokensRequest true "GUID пользователя"
// @Success 200 {object} TokensResponse
// @Failure 400 {string} string "invalid request"
// @Failure 500 {string} string "failed to generate/save token"
// @Router /tokens [post]
func (h *Handler) GetTokens(w http.ResponseWriter, r *http.Request) {
	var req GetTokensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	log.Printf("Received UserGUID: %s", req.UserGUID)

	accessToken, err := token.GenerateAccessToken(req.UserGUID, 15)
	if err != nil {
		http.Error(w, "failed to generate access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := token.GenerateRefreshToken(32)
	if err != nil {
		http.Error(w, "failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	userAgent := r.Header.Get("User-Agent")
	ip := r.RemoteAddr

	fmt.Println("req.UserGUID =", req.UserGUID)
	fmt.Println("refreshToken =", refreshToken)
	fmt.Println("userAgent =", userAgent)
	fmt.Println("ip =", ip)

	_, err = h.db.Exec(`
        INSERT INTO auth_sessions (user_guid, refresh_token, user_agent, ip, created_at, updated_at)
        VALUES ($1, $2, $3, $4, now(), now())
        ON CONFLICT (user_guid) DO UPDATE SET
        refresh_token = EXCLUDED.refresh_token,
        user_agent = EXCLUDED.user_agent,
        ip = EXCLUDED.ip,
        updated_at = now()
    `, req.UserGUID, refreshToken, userAgent, ip)

	if err != nil {
		fmt.Println("DB error:", err)
		http.Error(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	resp := TokensResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RefreshTokens godoc
// @Summary Обновление пары токенов
// @Description Обновляет access и refresh токены, если соблюдены условия (валидность refresh, совпадение User-Agent, etc.)
// @Tags auth
// @Accept json
// @Produce json
// @Param input body map[string]string true "GUID и refresh_token"
// @Success 200 {object} TokensResponse
// @Failure 400 {string} string "invalid request"
// @Failure 401 {string} string "unauthorized"
// @Failure 500 {string} string "internal error"
// @Router /tokens/refresh [post]
func (h *Handler) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	type RefreshRequest struct {
		UserGUID     string `json:"user_guid"`
		RefreshToken string `json:"refresh_token"`
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	currentUserAgent := r.Header.Get("User-Agent")
	currentIP := r.RemoteAddr

	var storedRefreshToken, storedUserAgent, storedIP string
	err := h.db.QueryRow(`SELECT refresh_token, user_agent, ip FROM auth_sessions WHERE user_guid = $1`, req.UserGUID).Scan(&storedRefreshToken, &storedUserAgent, &storedIP)
	if err == sql.ErrNoRows {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if req.RefreshToken != storedRefreshToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if currentUserAgent != storedUserAgent {
		_, _ = h.db.Exec(`DELETE FROM auth_sessions WHERE user_guid = $1`, req.UserGUID)
		http.Error(w, "unauthorized - user agent mismatch", http.StatusUnauthorized)
		return
	}

	if currentIP != storedIP {
		webhookURL := os.Getenv("WEBHOOK_URL")
		go notifyIPChange(webhookURL, req.UserGUID, currentIP)
	}

	newAccessToken, err := token.GenerateAccessToken(req.UserGUID, 15)
	if err != nil {
		http.Error(w, "failed to generate access token", http.StatusInternalServerError)
		return
	}

	newRefreshToken, err := token.GenerateRefreshToken(32)
	if err != nil {
		http.Error(w, "failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	_, err = h.db.Exec(`
		UPDATE auth_sessions SET refresh_token = $1, ip = $2, user_agent = $3, updated_at = now()
		WHERE user_guid = $4
	`, newRefreshToken, currentIP, currentUserAgent, req.UserGUID)
	if err != nil {
		http.Error(w, "failed to update session", http.StatusInternalServerError)
		return
	}

	resp := TokensResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func notifyIPChange(webhookURL, userGUID, newIP string) {
	if webhookURL == "" {
		return
	}
	payload := map[string]string{
		"user_guid": userGUID,
		"new_ip":    newIP,
	}
	body, _ := json.Marshal(payload)
	_, _ = http.Post(webhookURL, "application/json", bytes.NewBuffer(body))
}

// GetCurrentUser godoc
// @Summary Получить текущего пользователя
// @Description Возвращает GUID пользователя из access токена (защищённый маршрут)
// @Tags auth
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {string} string "unauthorized"
// @Router /me [get]
func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userGUID, err := middleware.GetUserGUIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"user_guid": userGUID})
}

// Logout godoc
// @Summary Деавторизация пользователя
// @Description Удаляет refresh токен пользователя и завершает сессию
// @Tags auth
// @Security ApiKeyAuth
// @Success 200 {string} string "logged out"
// @Failure 401 {string} string "missing or invalid token"
// @Failure 500 {string} string "logout failed"
// @Router /logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	tokenStr := strings.TrimPrefix(auth, "Bearer ")
	userGUID, err := token.ValidateAccessToken(tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	_, err = h.db.Exec("DELETE FROM auth_sessions WHERE user_guid = $1", userGUID)
	if err != nil {
		http.Error(w, "logout failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("logged out"))
}
