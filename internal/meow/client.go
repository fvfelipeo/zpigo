package meow

import (
	"database/sql"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"go.mau.fi/whatsmeow"
)

type ZPigoClient struct {
	WAClient *whatsmeow.Client

	SessionID string
	APIKey    string

	EventHandlerID uint32
	Subscriptions  []string

	DB *sql.DB

	HTTPClient *resty.Client

	IsActive    bool
	ConnectedAt *time.Time
	mu          sync.RWMutex

	KillChannel chan bool

	CacheManager *CacheManager
}

func NewZPigoClient(sessionID, apiKey string, waClient *whatsmeow.Client, db *sql.DB) *ZPigoClient {
	client := &ZPigoClient{
		WAClient:      waClient,
		SessionID:     sessionID,
		APIKey:        apiKey,
		DB:            db,
		HTTPClient:    NewHTTPClient(),
		IsActive:      false,
		Subscriptions: []string{},
		KillChannel:   make(chan bool, 1),
		CacheManager:  GetGlobalCache(),
	}

	if waClient != nil {
		client.EventHandlerID = waClient.AddEventHandler(client.EventHandler)
	}

	return client
}

func (zc *ZPigoClient) UpdateSubscriptions(subscriptions []string) {
	zc.mu.Lock()
	defer zc.mu.Unlock()
	zc.Subscriptions = subscriptions
}

func (zc *ZPigoClient) SetActive(active bool) {
	zc.mu.Lock()
	defer zc.mu.Unlock()
	zc.IsActive = active
	if active {
		now := time.Now()
		zc.ConnectedAt = &now
	} else {
		zc.ConnectedAt = nil
	}
}

func (zc *ZPigoClient) IsClientActive() bool {
	zc.mu.RLock()
	defer zc.mu.RUnlock()
	return zc.IsActive
}

func (zc *ZPigoClient) GetSubscriptions() []string {
	zc.mu.RLock()
	defer zc.mu.RUnlock()
	return append([]string{}, zc.Subscriptions...)
}

func (zc *ZPigoClient) GetSessionInfo() (*SessionInfo, bool) {
	cacheKey := BuildCacheKey(zc.APIKey, zc.SessionID)
	return zc.CacheManager.GetSessionInfo(cacheKey)
}

func (zc *ZPigoClient) UpdateSessionInfo(key, value string) {
	cacheKey := BuildCacheKey(zc.APIKey, zc.SessionID)
	zc.CacheManager.UpdateSessionInfo(cacheKey, key, value)
}

func (zc *ZPigoClient) SetProxy(proxyURL string) {
	if proxyURL != "" {
		zc.HTTPClient.SetProxy(proxyURL)
	}
}

func (zc *ZPigoClient) Disconnect() {
	zc.SetActive(false)
	if zc.WAClient != nil && zc.WAClient.IsConnected() {
		zc.WAClient.Disconnect()
	}
}

func (zc *ZPigoClient) Kill() {
	select {
	case zc.KillChannel <- true:
	default:
	}
}

func (zc *ZPigoClient) Cleanup() {
	zc.SetActive(false)
	if zc.WAClient != nil {
		if zc.EventHandlerID != 0 {
			zc.WAClient.RemoveEventHandler(zc.EventHandlerID)
		}
		if zc.WAClient.IsConnected() {
			zc.WAClient.Disconnect()
		}
	}
	close(zc.KillChannel)
}
