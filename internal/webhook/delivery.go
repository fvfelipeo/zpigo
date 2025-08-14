package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"zpigo/internal/logger"
)

func (wm *Manager) processDelivery(delivery *Delivery, workerLogger logger.Logger) {
	startTime := time.Now()
	delivery.Attempts++
	delivery.LastAttempt = startTime

	workerLogger.Debug("Processando delivery",
		"deliveryID", delivery.ID,
		"sessionID", delivery.SessionID,
		"attempt", delivery.Attempts,
		"url", delivery.URL)

	payloadBytes, err := json.Marshal(delivery.Payload)
	if err != nil {
		delivery.Status = string(StatusFailed)
		delivery.Error = fmt.Sprintf("Erro ao serializar payload: %v", err)
		workerLogger.Error("Erro ao serializar payload", "error", err, "deliveryID", delivery.ID)
		wm.incrementStat("total_failed")
		return
	}

	req := wm.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", "ZPigo-Webhook/1.0").
		SetBody(payloadBytes)

	config, exists := wm.GetConfig(delivery.SessionID)
	if exists && config.Headers != nil {
		for key, value := range config.Headers {
			req.SetHeader(key, value)
		}
	}

	if exists && config.Secret != "" {
		signature := wm.generateSignature(payloadBytes, config.Secret)
		req.SetHeader("X-Webhook-Signature", signature)
	}

	req.SetHeader("X-Webhook-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

	resp, err := req.Post(delivery.URL)
	duration := time.Since(startTime)
	delivery.Duration = duration

	if err != nil {
		delivery.Error = fmt.Sprintf("Erro de rede: %v", err)
		wm.handleDeliveryFailure(delivery, workerLogger)
		return
	}

	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		delivery.Status = string(StatusSuccess)
		workerLogger.Info("Webhook entregue com sucesso",
			"deliveryID", delivery.ID,
			"statusCode", resp.StatusCode(),
			"duration", duration)
		wm.incrementStat("total_success")
	} else {
		delivery.Error = fmt.Sprintf("Status code inválido: %d", resp.StatusCode())
		wm.handleDeliveryFailure(delivery, workerLogger)
	}
}

func (wm *Manager) handleDeliveryFailure(delivery *Delivery, workerLogger logger.Logger) {
	workerLogger.Warn("Falha na entrega de webhook",
		"deliveryID", delivery.ID,
		"attempt", delivery.Attempts,
		"error", delivery.Error)

	if delivery.Attempts < delivery.MaxRetries {
		backoffDelay := time.Duration(delivery.Attempts) * 5 * time.Second
		delivery.NextRetry = time.Now().Add(backoffDelay)

		workerLogger.Info("Agendando retry",
			"deliveryID", delivery.ID,
			"nextRetry", delivery.NextRetry,
			"backoffDelay", backoffDelay)

		go func() {
			time.Sleep(backoffDelay)
			select {
			case wm.deliveryQueue <- delivery:
				wm.incrementStat("total_retries")
			default:
				workerLogger.Warn("Fila cheia, descartando retry", "deliveryID", delivery.ID)
			}
		}()
	} else {
		delivery.Status = string(StatusExpired)
		workerLogger.Error("Delivery expirada após máximo de tentativas",
			"deliveryID", delivery.ID,
			"attempts", delivery.Attempts)
		wm.incrementStat("total_failed")
	}
}

func (wm *Manager) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

func (wm *Manager) SendTestWebhook(sessionID, url string) error {
	testPayload := &Payload{
		Type:      "test",
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Event: map[string]interface{}{
			"message": "Este é um webhook de teste do ZPigo",
			"version": "1.0",
		},
		Data: map[string]interface{}{
			"test": true,
		},
	}

	payloadBytes, err := json.Marshal(testPayload)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload de teste: %v", err)
	}

	resp, err := wm.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", "ZPigo-Webhook/1.0").
		SetHeader("X-Webhook-Test", "true").
		SetBody(payloadBytes).
		Post(url)

	if err != nil {
		return fmt.Errorf("erro ao enviar webhook de teste: %v", err)
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return fmt.Errorf("webhook de teste falhou com status %d", resp.StatusCode())
	}

	wm.logger.Info("Webhook de teste enviado com sucesso",
		"sessionID", sessionID,
		"url", url,
		"statusCode", resp.StatusCode())

	return nil
}

func (wm *Manager) ValidateWebhookEndpoint(url string) (*Response, error) {
	startTime := time.Now()

	resp, err := wm.httpClient.R().
		SetHeader("User-Agent", "ZPigo-Webhook/1.0").
		SetHeader("X-Webhook-Validation", "true").
		Get(url)

	duration := time.Since(startTime)

	if err != nil {
		return &Response{
			Duration: duration,
			Error:    err.Error(),
		}, err
	}

	headers := make(map[string]string)
	for key, values := range resp.Header() {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return &Response{
		StatusCode: resp.StatusCode(),
		Headers:    headers,
		Body:       string(resp.Body()),
		Duration:   duration,
	}, nil
}

func (wm *Manager) GetDeliveryHistory(sessionID string, limit int) ([]*Delivery, error) {
	return []*Delivery{}, nil
}

func (wm *Manager) RetryFailedDeliveries(sessionID string) error {
	wm.logger.Info("Retry de deliveries falhadas solicitado", "sessionID", sessionID)
	return nil
}
