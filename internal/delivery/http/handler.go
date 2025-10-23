package delivery

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/CAATHARSIS/courier-bot/internal/service/assignment"
)

type WebhookHandler struct {
	assignmentService *assignment.Service
	webhookSecret     string
	log               *slog.Logger
}

type NewOrderWebhook struct {
	OrderID   int    `json:"order_id" validate:"required,min=1"`
	Signature string `json:"signature,omitempty"`
}

type WebHookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func NewWebhookHandler(assignmentService *assignment.Service, webhookSecret string, log *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		assignmentService: assignmentService,
		webhookSecret:     webhookSecret,
		log:               log,
	}
}

func (h *WebhookHandler) HandleNewOrderWebhook(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.log.Warn("Invalid HTTP mehtod for webhook", "Method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("Failed to read request body", "Error", err)
		h.sendErrorResponse(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var webhook NewOrderWebhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		h.log.Error("Failed to decode webhook payload", "Error", err)
		h.sendErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if webhook.OrderID <= 0 {
		h.log.Warn("Invalid order ID", "orderID", webhook.OrderID)
		h.sendErrorResponse(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	if h.webhookSecret != "" {
		validSignature := h.verifySignature(r)

		if !validSignature {
			h.log.Warn("Invalid webhook signature for order", "orderID", webhook.OrderID)
			h.sendErrorResponse(w, "Invalid signature", http.StatusUnauthorized)
		}
	}

	h.log.Info("Received new order webhook", "orderID", webhook.OrderID)

	h.sendSuccessResponse(w, "Order processing started")

	go h.processOrderAssignment(ctx, webhook.OrderID)
}

func (h *WebhookHandler) processOrderAssignment(ctx context.Context, orderID int) {
	startTime := time.Now()
	h.log.Info("Starting async order assignment processing", "orderID", orderID)

	if err := h.assignmentService.ProcessNewOrder(ctx, orderID); err != nil {
		h.log.Error("Failed to process order", "orderID", orderID, "Error", err)
		return
	}

	processingTime := time.Since(startTime)
	h.log.Info("Order assignment processing completed", "orderID", orderID, "time", processingTime)
}

func (h *WebhookHandler) verifySignature(r *http.Request) bool {
	if h.webhookSecret == "" {
		h.log.Debug("Webhook secret not set, signature verifacation disabled")
		return true
	}

	signatureHeader := r.Header.Get("X-Signature")
	if signatureHeader == "" {
		h.log.Warn("Missing X-Signature header in webhook request")
		return false
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("Failed to read request body for signature verification", "Error", err)
		return false
	}

	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	expectedSignature, err := h.computeHMACSHA256(bodyBytes, h.webhookSecret)
	if err != nil {
		h.log.Error("Failed to compute HMAC signature", "Error", err)
		return false
	}

	isValid := hmac.Equal([]byte(signatureHeader), []byte(expectedSignature))

	if !isValid {
		h.log.Warn("Invalid webhook signature", "Expected", expectedSignature, "Received", signatureHeader)
	}

	return isValid
}

func (h *WebhookHandler) computeHMACSHA256(data []byte, secret string) (string, error) {
	mac := hmac.New(sha256.New, []byte(secret))

	if _, err := mac.Write(data); err != nil {
		return "", fmt.Errorf("failed to write data to HMAC: %v", err)
	}

	signature := mac.Sum(nil)
	return hex.EncodeToString(signature), nil
}

func (h *WebhookHandler) sendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := WebHookResponse{
		Success: false,
		Error:   errorMsg,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Failed to encode error response", "Error", err)
	}
}

func (h *WebhookHandler) sendSuccessResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := WebHookResponse{
		Success: true,
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Failed to encode success response", "Error", err)
	}
}
