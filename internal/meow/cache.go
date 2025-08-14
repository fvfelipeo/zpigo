package meow

import (
	"time"

	"github.com/patrickmn/go-cache"
)

type SessionInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	JID     string `json:"jid"`
	Webhook string `json:"webhook"`
	APIKey  string `json:"apikey"`
	Events  string `json:"events"`
	Proxy   string `json:"proxy"`
	QRCode  string `json:"qrcode"`
	Phone   string `json:"phone"`
	Status  string `json:"status"`
}

func (s *SessionInfo) Get(key string) string {
	switch key {
	case "Id":
		return s.ID
	case "Name":
		return s.Name
	case "Jid":
		return s.JID
	case "Webhook":
		return s.Webhook
	case "APIKey":
		return s.APIKey
	case "Events":
		return s.Events
	case "Proxy":
		return s.Proxy
	case "Qrcode":
		return s.QRCode
	case "Phone":
		return s.Phone
	case "Status":
		return s.Status
	default:
		return ""
	}
}

func (s *SessionInfo) Set(key, value string) {
	switch key {
	case "Id":
		s.ID = value
	case "Name":
		s.Name = value
	case "Jid":
		s.JID = value
	case "Webhook":
		s.Webhook = value
	case "APIKey":
		s.APIKey = value
	case "Events":
		s.Events = value
	case "Proxy":
		s.Proxy = value
	case "Qrcode":
		s.QRCode = value
	case "Phone":
		s.Phone = value
	case "Status":
		s.Status = value
	}
}

type CacheManager struct {
	cache *cache.Cache
}

func NewCacheManager() *CacheManager {
	c := cache.New(24*time.Hour, 1*time.Hour)
	return &CacheManager{
		cache: c,
	}
}

func (cm *CacheManager) SetSessionInfo(sessionID string, sessionInfo *SessionInfo) {
	cm.cache.Set(sessionID, sessionInfo, cache.NoExpiration)
}

func (cm *CacheManager) GetSessionInfo(sessionID string) (*SessionInfo, bool) {
	if item, found := cm.cache.Get(sessionID); found {
		if sessionInfo, ok := item.(*SessionInfo); ok {
			return sessionInfo, true
		}
	}
	return nil, false
}

func (cm *CacheManager) UpdateSessionInfo(sessionID, key, value string) bool {
	if sessionInfo, found := cm.GetSessionInfo(sessionID); found {
		sessionInfo.Set(key, value)
		cm.SetSessionInfo(sessionID, sessionInfo)
		return true
	}
	return false
}

func (cm *CacheManager) DeleteSessionInfo(sessionID string) {
	cm.cache.Delete(sessionID)
}

func (cm *CacheManager) ClearCache() {
	cm.cache.Flush()
}

func (cm *CacheManager) GetCacheStats() (int, int) {
	return cm.cache.ItemCount(), len(cm.cache.Items())
}

func (cm *CacheManager) SetWithExpiration(key string, value interface{}, duration time.Duration) {
	cm.cache.Set(key, value, duration)
}

func (cm *CacheManager) Get(key string) (interface{}, bool) {
	return cm.cache.Get(key)
}

func (cm *CacheManager) Set(key string, value interface{}) {
	cm.cache.Set(key, value, cache.NoExpiration)
}

func (cm *CacheManager) Delete(key string) {
	cm.cache.Delete(key)
}

var GlobalCacheManager *CacheManager

func InitGlobalCache() {
	GlobalCacheManager = NewCacheManager()
}

func GetGlobalCache() *CacheManager {
	if GlobalCacheManager == nil {
		InitGlobalCache()
	}
	return GlobalCacheManager
}
