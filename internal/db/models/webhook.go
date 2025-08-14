package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Webhook struct {
	bun.BaseModel `bun:"table:webhooks,alias:w"`

	ID        string `json:"id" bun:"id,pk,type:varchar(255)"`
	SessionID string `json:"sessionId" bun:"sessionId,notnull,type:varchar(255)"`
	URL       string `json:"url" bun:"url,notnull,type:varchar(500)"`
	Events    string `json:"events" bun:"events,type:text"`

	CreatedAt time.Time `json:"createdAt" bun:"createdAt,nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `json:"updatedAt" bun:"updatedAt,nullzero,notnull,default:current_timestamp"`

	Session *Session `json:"session,omitempty" bun:"rel:belongs-to,join:sessionId=id"`
}

func (Webhook) TableName() string {
	return "webhooks"
}

func (w *Webhook) BeforeAppendModel(query bun.Query) error {
	switch query.(type) {
	case *bun.UpdateQuery:
		w.UpdatedAt = time.Now()
	}
	return nil
}
