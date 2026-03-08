package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rerolldrinks/messageservice/config"
	"github.com/rerolldrinks/messageservice/helpers"
	"github.com/rerolldrinks/messageservice/models"
)

type WebhookService interface {
	TriggerWebhook(ctx context.Context, msg models.MessageRecord) error
}

type webhookService struct {
	settings      *config.MessageSettings
	broadcastFunc func(ctx context.Context, payload string) error
}

func NewWebhookService(
	settings *config.MessageSettings,
	broadcastFunc func(ctx context.Context, payload string) error,
) WebhookService {
	return &webhookService{settings: settings, broadcastFunc: broadcastFunc}
}

func (w *webhookService) TriggerWebhook(ctx context.Context, msg models.MessageRecord) error {
	if w.settings.EncryptionKey == "" || w.settings.RequestKey == "" {
		return errors.New("EncryptionKey or RequestKey is not configured")
	}

	payload, err := w.createPayload(msg)
	if err != nil {
		return fmt.Errorf("failed to create payload: %w", err)
	}

	if err := w.broadcastFunc(ctx, payload); err != nil {
		return fmt.Errorf("failed to broadcast message: %w", err)
	}

	return nil
}

func (w *webhookService) createPayload(msg models.MessageRecord) (string, error) {
	req := models.DecryptedMessageRequest{
		Message:    msg,
		RequestKey: w.settings.RequestKey,
	}

	encrypted, err := helpers.EncryptMessage(req, w.settings.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}

	jsonBytes, err := json.Marshal(models.EncryptedMessageRequest{EncodedRequest: encrypted})
	if err != nil {
		return "", fmt.Errorf("failed to marshal encrypted payload: %w", err)
	}

	return string(jsonBytes), nil
}
