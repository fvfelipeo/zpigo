package webhook

import (
	"time"
)

type Config struct {
	URL        string            `json:"url"`
	Events     []string          `json:"events"`
	Headers    map[string]string `json:"headers,omitempty"`
	Timeout    time.Duration     `json:"timeout"`
	MaxRetries int               `json:"max_retries"`
	RetryDelay time.Duration     `json:"retry_delay"`
	Enabled    bool              `json:"enabled"`
	Secret     string            `json:"secret,omitempty"`
}

type Payload struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"sessionId"`
	Timestamp int64                  `json:"timestamp"`
	Event     interface{}            `json:"event"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type Delivery struct {
	ID          string        `json:"id"`
	SessionID   string        `json:"sessionId"`
	URL         string        `json:"url"`
	Payload     interface{}   `json:"payload"`
	Attempts    int           `json:"attempts"`
	MaxRetries  int           `json:"max_retries"`
	LastAttempt time.Time     `json:"last_attempt"`
	NextRetry   time.Time     `json:"next_retry"`
	Status      string        `json:"status"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
}

type DeliveryStatus string

const (
	StatusPending DeliveryStatus = "pending"
	StatusSuccess DeliveryStatus = "success"
	StatusFailed  DeliveryStatus = "failed"
	StatusExpired DeliveryStatus = "expired"
)

type EventType string

const (
	EventConnected                   EventType = "Connected"
	EventDisconnected                EventType = "Disconnected"
	EventLoggedOut                   EventType = "LoggedOut"
	EventPairSuccess                 EventType = "PairSuccess"
	EventPairError                   EventType = "PairError"
	EventQR                          EventType = "QR"
	EventQRScannedWithoutMultidevice EventType = "QRScannedWithoutMultidevice"
	EventStreamReplaced              EventType = "StreamReplaced"
	EventStreamError                 EventType = "StreamError"
	EventConnectFailure              EventType = "ConnectFailure"
	EventClientOutdated              EventType = "ClientOutdated"
	EventTemporaryBan                EventType = "TemporaryBan"
	EventCATRefreshError             EventType = "CATRefreshError"
	EventKeepAliveTimeout            EventType = "KeepAliveTimeout"
	EventKeepAliveRestored           EventType = "KeepAliveRestored"
	EventManualLoginReconnect        EventType = "ManualLoginReconnect"

	EventMessage              EventType = "Message"
	EventFBMessage            EventType = "FBMessage"
	EventReceipt              EventType = "Receipt"
	EventUndecryptableMessage EventType = "UndecryptableMessage"
	EventMediaRetry           EventType = "MediaRetry"
	EventMediaRetryError      EventType = "MediaRetryError"

	EventPresence     EventType = "Presence"
	EventChatPresence EventType = "ChatPresence"

	EventGroupInfo   EventType = "GroupInfo"
	EventJoinedGroup EventType = "JoinedGroup"

	EventContact      EventType = "Contact"
	EventPushName     EventType = "PushName"
	EventBusinessName EventType = "BusinessName"
	EventPicture      EventType = "Picture"
	EventUserAbout    EventType = "UserAbout"

	EventArchive        EventType = "Archive"
	EventPin            EventType = "Pin"
	EventMute           EventType = "Mute"
	EventStar           EventType = "Star"
	EventMarkChatAsRead EventType = "MarkChatAsRead"
	EventDeleteChat     EventType = "DeleteChat"
	EventClearChat      EventType = "ClearChat"
	EventDeleteForMe    EventType = "DeleteForMe"

	EventLabelEdit               EventType = "LabelEdit"
	EventLabelAssociationChat    EventType = "LabelAssociationChat"
	EventLabelAssociationMessage EventType = "LabelAssociationMessage"

	EventPrivacySettings       EventType = "PrivacySettings"
	EventPushNameSetting       EventType = "PushNameSetting"
	EventUnarchiveChatsSetting EventType = "UnarchiveChatsSetting"

	EventHistorySync          EventType = "HistorySync"
	EventAppState             EventType = "AppState"
	EventAppStateSyncComplete EventType = "AppStateSyncComplete"
	EventOfflineSyncPreview   EventType = "OfflineSyncPreview"
	EventOfflineSyncCompleted EventType = "OfflineSyncCompleted"

	EventCallOffer        EventType = "CallOffer"
	EventCallOfferNotice  EventType = "CallOfferNotice"
	EventCallAccept       EventType = "CallAccept"
	EventCallPreAccept    EventType = "CallPreAccept"
	EventCallReject       EventType = "CallReject"
	EventCallTerminate    EventType = "CallTerminate"
	EventCallRelayLatency EventType = "CallRelayLatency"
	EventCallTransport    EventType = "CallTransport"
	EventUnknownCallEvent EventType = "UnknownCallEvent"

	EventNewsletterJoin       EventType = "NewsletterJoin"
	EventNewsletterLeave      EventType = "NewsletterLeave"
	EventNewsletterLiveUpdate EventType = "NewsletterLiveUpdate"
	EventNewsletterMuteChange EventType = "NewsletterMuteChange"

	EventBlocklist EventType = "Blocklist"

	EventIdentityChange EventType = "IdentityChange"

	EventUserStatusMute EventType = "UserStatusMute"

	EventAll EventType = "All"
)

type Response struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Duration   time.Duration     `json:"duration"`
	Error      string            `json:"error,omitempty"`
}

type Stats struct {
	TotalSent      int64 `json:"total_sent"`
	TotalSuccess   int64 `json:"total_success"`
	TotalFailed    int64 `json:"total_failed"`
	TotalRetries   int64 `json:"total_retries"`
	AverageLatency int64 `json:"average_latency_ms"`
	QueueSize      int   `json:"queue_size"`
}

type Filter struct {
	Events    []string `json:"events,omitempty"`
	SessionID string   `json:"session_id,omitempty"`
	FromMe    *bool    `json:"from_me,omitempty"`
	IsGroup   *bool    `json:"is_group,omitempty"`
}
