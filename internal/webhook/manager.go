package webhook

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"

	"zpigo/internal/logger"
)

type Manager struct {
	configs map[string]*Config
	mu      sync.RWMutex

	httpClient *resty.Client

	deliveryQueue chan *Delivery

	workers    int
	stopChan   chan bool
	workerWG   sync.WaitGroup

	logger logger.Logger

	globalConfig *Config

	stats Stats
	statsMu sync.RWMutex
}

func NewManager(workers int) *Manager {
	wm := &Manager{
		configs:       make(map[string]*Config),
		httpClient:    newHTTPClient(),
		deliveryQueue: make(chan *Delivery, 1000),
		workers:       workers,
		stopChan:      make(chan bool),
		logger:        logger.NewForComponent("WebhookManager"),
	}

	wm.startWorkers()

	return wm
}

func newHTTPClient() *resty.Client {
	client := resty.New()
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(15))
	client.SetTimeout(10 * time.Second)
	client.SetRetryCount(0)
	
	return client
}

func (wm *Manager) SetConfig(sessionID string, config *Config) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if !isValidURL(config.URL) {
		wm.logger.Warn("URL de webhook inválida", "sessionID", sessionID, "url", config.URL)
		return fmt.Errorf("URL de webhook inválida: %s", config.URL)
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}

	wm.configs[sessionID] = config
	wm.logger.Info("Webhook configurado", "sessionID", sessionID, "url", config.URL, "events", len(config.Events))
	
	return nil
}

func (wm *Manager) GetConfig(sessionID string) (*Config, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	config, exists := wm.configs[sessionID]
	return config, exists
}

func (wm *Manager) DeleteConfig(sessionID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	delete(wm.configs, sessionID)
	wm.logger.Info("Webhook removido", "sessionID", sessionID)
}

func (wm *Manager) SetGlobalConfig(config *Config) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	if !isValidURL(config.URL) {
		return fmt.Errorf("URL de webhook global inválida: %s", config.URL)
	}
	
	wm.globalConfig = config
	wm.logger.Info("Webhook global configurado", "url", config.URL)
	
	return nil
}

func (wm *Manager) Send(sessionID string, eventType EventType, eventData interface{}, additionalData map[string]interface{}) {
	config, hasSessionConfig := wm.GetConfig(sessionID)
	
	if hasSessionConfig && config.Enabled && wm.shouldSendEvent(config.Events, string(eventType)) {
		wm.queueDelivery(sessionID, config, eventType, eventData, additionalData)
	}

	wm.mu.RLock()
	globalConfig := wm.globalConfig
	wm.mu.RUnlock()

	if globalConfig != nil && globalConfig.Enabled && wm.shouldSendEvent(globalConfig.Events, string(eventType)) {
		wm.queueDelivery("global", globalConfig, eventType, eventData, additionalData)
	}
}

func (wm *Manager) shouldSendEvent(configuredEvents []string, eventType string) bool {
	if len(configuredEvents) == 0 {
		return false
	}

	for _, event := range configuredEvents {
		if event == "All" || event == eventType {
			return true
		}
	}

	return false
}

func (wm *Manager) queueDelivery(sessionID string, config *Config, eventType EventType, eventData interface{}, additionalData map[string]interface{}) {
	payload := &Payload{
		Type:      string(eventType),
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Event:     eventData,
		Data:      additionalData,
	}

	delivery := &Delivery{
		ID:         fmt.Sprintf("%s-%d", sessionID, time.Now().UnixNano()),
		SessionID:  sessionID,
		URL:        config.URL,
		Payload:    payload,
		Attempts:   0,
		MaxRetries: config.MaxRetries,
		Status:     string(StatusPending),
	}

	select {
	case wm.deliveryQueue <- delivery:
		wm.logger.Debug("Webhook enfileirado", "sessionID", sessionID, "eventType", eventType, "url", config.URL)
		wm.incrementStat("total_sent")
	default:
		wm.logger.Warn("Fila de webhooks cheia, descartando delivery", "sessionID", sessionID, "eventType", eventType)
	}
}

func (wm *Manager) GetStats() Stats {
	wm.statsMu.RLock()
	defer wm.statsMu.RUnlock()
	
	stats := wm.stats
	stats.QueueSize = len(wm.deliveryQueue)
	
	return stats
}

func (wm *Manager) incrementStat(stat string) {
	wm.statsMu.Lock()
	defer wm.statsMu.Unlock()
	
	switch stat {
	case "total_sent":
		wm.stats.TotalSent++
	case "total_success":
		wm.stats.TotalSuccess++
	case "total_failed":
		wm.stats.TotalFailed++
	case "total_retries":
		wm.stats.TotalRetries++
	}
}

func (wm *Manager) startWorkers() {
	for i := 0; i < wm.workers; i++ {
		wm.workerWG.Add(1)
		go wm.worker(i)
	}
	wm.logger.Info("Workers de webhook iniciados", "count", wm.workers)
}

func (wm *Manager) worker(id int) {
	defer wm.workerWG.Done()
	workerLogger := wm.logger.With("worker", id)

	for {
		select {
		case delivery := <-wm.deliveryQueue:
			wm.processDelivery(delivery, workerLogger)
		case <-wm.stopChan:
			workerLogger.Info("Worker de webhook parado")
			return
		}
	}
}

func (wm *Manager) Stop() {
	wm.logger.Info("Parando gerenciador de webhooks")
	
	for i := 0; i < wm.workers; i++ {
		wm.stopChan <- true
	}
	
	wm.workerWG.Wait()
	
	close(wm.deliveryQueue)
	close(wm.stopChan)
	
	wm.logger.Info("Gerenciador de webhooks parado")
}

func isValidURL(url string) bool {
	return url != "" && (len(url) > 7) && (url[:7] == "http://" || url[:8] == "https://")
}
