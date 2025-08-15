package models

import (
	"time"
)

type Webhook struct {
	ID        string `json:"id" db:"id"`
	SessionID string `json:"sessionId" db:"sessionid"`
	URL       string `json:"url" db:"url"`
	Events    string `json:"events" db:"events"`

	CreatedAt time.Time `json:"createdAt" db:"createdat"`
	UpdatedAt time.Time `json:"updatedAt" db:"updatedat"`

	Session *Session `json:"session,omitempty"`
}

func (Webhook) TableName() string {
	return "webhooks"
}
