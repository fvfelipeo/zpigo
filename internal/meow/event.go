package meow

import (
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types/events"

	"zpigo/internal/logger"
	"zpigo/internal/webhook"
)

func (zc *ZPigoClient) EventHandler(rawEvt interface{}) {
	zc.mu.RLock()
	defer zc.mu.RUnlock()

	if !zc.IsActive {
		return
	}

	eventLogger := logger.WithComponent("EventHandler").With("sessionID", zc.SessionID)

	postmap := make(map[string]interface{})
	postmap["event"] = rawEvt
	postmap["sessionId"] = zc.SessionID
	postmap["timestamp"] = time.Now().Unix()

	shouldCallWebhook := false
	eventType := ""

	switch evt := rawEvt.(type) {
	case *events.Connected:
		eventType = string(webhook.EventConnected)
		shouldCallWebhook = true
		eventLogger.Info("Cliente conectado")
		zc.handleConnectedEvent()

	case *events.Disconnected:
		eventType = string(webhook.EventDisconnected)
		shouldCallWebhook = true
		eventLogger.Info("Cliente desconectado")
		zc.handleDisconnectedEvent()

	case *events.PairSuccess:
		eventType = string(webhook.EventPairSuccess)
		shouldCallWebhook = true
		eventLogger.Info("Pareamento realizado com sucesso", "jid", evt.ID.String())
		zc.handlePairSuccessEvent(evt)

	case *events.PairError:
		eventType = string(webhook.EventPairError)
		shouldCallWebhook = true
		eventLogger.Error("Erro no pareamento", "error", evt.Error)
		zc.handlePairErrorEvent(evt, postmap)

	case *events.QR:
		eventType = string(webhook.EventQR)
		shouldCallWebhook = true
		eventLogger.Info("QR Code gerado", "codes", len(evt.Codes))
		zc.handleQREvent(evt, postmap)

	case *events.LoggedOut:
		eventType = string(webhook.EventLoggedOut)
		shouldCallWebhook = true
		eventLogger.Warn("Cliente deslogado", "reason", evt.Reason)
		zc.handleLoggedOutEvent(evt)

	case *events.StreamReplaced:
		eventType = string(webhook.EventStreamReplaced)
		shouldCallWebhook = true
		eventLogger.Warn("Stream substituído")
		zc.handleStreamReplacedEvent()

	case *events.StreamError:
		eventType = string(webhook.EventStreamError)
		shouldCallWebhook = true
		eventLogger.Error("Erro de stream", "code", evt.Code)
		zc.handleStreamErrorEvent(evt, postmap)

	case *events.ConnectFailure:
		eventType = string(webhook.EventConnectFailure)
		shouldCallWebhook = true
		eventLogger.Error("Falha na conexão", "reason", evt.Reason, "message", evt.Message)
		zc.handleConnectFailureEvent(evt, postmap)

	case *events.ClientOutdated:
		eventType = string(webhook.EventClientOutdated)
		shouldCallWebhook = true
		eventLogger.Warn("Cliente desatualizado")

	case *events.TemporaryBan:
		eventType = string(webhook.EventTemporaryBan)
		shouldCallWebhook = true
		eventLogger.Warn("Ban temporário", "code", evt.Code, "expire", evt.Expire)
		zc.handleTemporaryBanEvent(evt, postmap)

	case *events.Message:
		eventType = string(webhook.EventMessage)
		shouldCallWebhook = true
		eventLogger.Debug("Mensagem recebida", "from", evt.Info.Sender.String(), "messageID", evt.Info.ID)
		zc.handleMessageEvent(evt, postmap)

	case *events.FBMessage:
		eventType = string(webhook.EventFBMessage)
		shouldCallWebhook = true
		eventLogger.Debug("Mensagem FB recebida", "from", evt.Info.Sender.String(), "messageID", evt.Info.ID)
		zc.handleFBMessageEvent(evt, postmap)

	case *events.Receipt:
		eventType = string(webhook.EventReceipt)
		shouldCallWebhook = true
		eventLogger.Debug("Recibo recebido", "type", evt.Type, "messageIDs", len(evt.MessageIDs))
		zc.handleReceiptEvent(evt, postmap)

	case *events.UndecryptableMessage:
		eventType = string(webhook.EventUndecryptableMessage)
		shouldCallWebhook = true
		eventLogger.Warn("Mensagem não descriptografável", "from", evt.Info.Sender.String())
		zc.handleUndecryptableMessageEvent(evt, postmap)

	case *events.Presence:
		eventType = string(webhook.EventPresence)
		shouldCallWebhook = true
		eventLogger.Debug("Presença atualizada", "from", evt.From.String(), "unavailable", evt.Unavailable)
		zc.handlePresenceEvent(evt, postmap)

	case *events.ChatPresence:
		eventType = string(webhook.EventChatPresence)
		shouldCallWebhook = true
		eventLogger.Debug("Presença do chat atualizada", "chat", evt.MessageSource.Chat.String(), "state", evt.State)
		zc.handleChatPresenceEvent(evt, postmap)

	case *events.GroupInfo:
		eventType = string(webhook.EventGroupInfo)
		shouldCallWebhook = true
		eventLogger.Info("Informações do grupo atualizadas", "jid", evt.JID.String())
		zc.handleGroupInfoEvent(evt, postmap)

	case *events.JoinedGroup:
		eventType = string(webhook.EventJoinedGroup)
		shouldCallWebhook = true
		eventLogger.Info("Entrou no grupo", "jid", evt.JID.String(), "reason", evt.Reason)
		zc.handleJoinedGroupEvent(evt, postmap)

	case *events.Contact:
		eventType = string(webhook.EventContact)
		shouldCallWebhook = true
		eventLogger.Debug("Contato atualizado", "jid", evt.JID.String())
		zc.handleContactEvent(evt, postmap)

	case *events.PushName:
		eventType = string(webhook.EventPushName)
		shouldCallWebhook = true
		eventLogger.Debug("Push name atualizado", "jid", evt.JID.String(), "newName", evt.NewPushName)
		zc.handlePushNameEvent(evt, postmap)

	case *events.BusinessName:
		eventType = string(webhook.EventBusinessName)
		shouldCallWebhook = true
		eventLogger.Debug("Nome comercial atualizado", "jid", evt.JID.String(), "newName", evt.NewBusinessName)
		zc.handleBusinessNameEvent(evt, postmap)

	case *events.Picture:
		eventType = string(webhook.EventPicture)
		shouldCallWebhook = true
		eventLogger.Debug("Foto atualizada", "jid", evt.JID.String(), "remove", evt.Remove)
		zc.handlePictureEvent(evt, postmap)

	default:
		eventLogger.Debug("Evento não tratado", "type", fmt.Sprintf("%T", rawEvt))
		return
	}

	if shouldCallWebhook && zc.shouldSendEvent(eventType) {
		postmap["type"] = eventType
		eventLogger.Debug("Enviando webhook", "eventType", eventType)
		go zc.callWebhook(postmap)
	}
}

func (zc *ZPigoClient) shouldSendEvent(eventType string) bool {
	if len(zc.Subscriptions) == 0 {
		return false
	}

	for _, sub := range zc.Subscriptions {
		if sub == "All" || sub == eventType {
			return true
		}
	}

	return false
}



func (zc *ZPigoClient) handleConnectedEvent() {
	zc.SetActive(true)
	zc.UpdateSessionInfo("Status", "connected")
}

func (zc *ZPigoClient) handleDisconnectedEvent() {
	zc.SetActive(false)
	zc.UpdateSessionInfo("Status", "disconnected")
}

func (zc *ZPigoClient) handlePairErrorEvent(evt *events.PairError, postmap map[string]interface{}) {
	postmap["error"] = evt.Error.Error()
	postmap["jid"] = evt.ID.String()
	postmap["lid"] = evt.LID.String()
	postmap["businessName"] = evt.BusinessName
	postmap["platform"] = evt.Platform
}

func (zc *ZPigoClient) handleQREvent(evt *events.QR, postmap map[string]interface{}) {
	postmap["codes"] = evt.Codes
	if len(evt.Codes) > 0 {
		zc.UpdateSessionInfo("Qrcode", evt.Codes[0])
	}
}

func (zc *ZPigoClient) handleStreamReplacedEvent() {
	zc.SetActive(false)
	zc.UpdateSessionInfo("Status", "stream_replaced")
}

func (zc *ZPigoClient) handleStreamErrorEvent(evt *events.StreamError, postmap map[string]interface{}) {
	postmap["code"] = evt.Code
	postmap["raw"] = evt.Raw
}

func (zc *ZPigoClient) handleConnectFailureEvent(evt *events.ConnectFailure, postmap map[string]interface{}) {
	postmap["reason"] = evt.Reason
	postmap["message"] = evt.Message
	postmap["raw"] = evt.Raw
}

func (zc *ZPigoClient) handleTemporaryBanEvent(evt *events.TemporaryBan, postmap map[string]interface{}) {
	postmap["code"] = evt.Code
	postmap["expire"] = evt.Expire.String()
}

func (zc *ZPigoClient) handlePairSuccessEvent(evt *events.PairSuccess) {
	if evt.ID.String() != "" {
		zc.UpdateSessionInfo("Jid", evt.ID.String())
	}
}


func (zc *ZPigoClient) handleMessageEvent(evt *events.Message, postmap map[string]interface{}) {
	postmap["messageId"] = evt.Info.ID
	postmap["from"] = evt.Info.Sender.String()
	postmap["timestamp"] = evt.Info.Timestamp.Unix()
	postmap["isFromMe"] = evt.Info.IsFromMe
	postmap["isGroup"] = evt.Info.IsGroup
	postmap["isEphemeral"] = evt.IsEphemeral
	postmap["isViewOnce"] = evt.IsViewOnce
	postmap["isEdit"] = evt.IsEdit
	postmap["retryCount"] = evt.RetryCount

}

func (zc *ZPigoClient) handleFBMessageEvent(evt *events.FBMessage, postmap map[string]interface{}) {
	postmap["messageId"] = evt.Info.ID
	postmap["from"] = evt.Info.Sender.String()
	postmap["timestamp"] = evt.Info.Timestamp.Unix()
	postmap["isFromMe"] = evt.Info.IsFromMe
	postmap["isGroup"] = evt.Info.IsGroup
	postmap["retryCount"] = evt.RetryCount
}

func (zc *ZPigoClient) handleUndecryptableMessageEvent(evt *events.UndecryptableMessage, postmap map[string]interface{}) {
	postmap["messageId"] = evt.Info.ID
	postmap["from"] = evt.Info.Sender.String()
	postmap["timestamp"] = evt.Info.Timestamp.Unix()
	postmap["isUnavailable"] = evt.IsUnavailable
	postmap["unavailableType"] = string(evt.UnavailableType)
	postmap["decryptFailMode"] = string(evt.DecryptFailMode)
}

func (zc *ZPigoClient) handleReceiptEvent(evt *events.Receipt, postmap map[string]interface{}) {
	postmap["messageIds"] = evt.MessageIDs
	postmap["receiptType"] = string(evt.Type)
	postmap["timestamp"] = evt.Timestamp.Unix()
}

func (zc *ZPigoClient) handlePresenceEvent(evt *events.Presence, postmap map[string]interface{}) {
	postmap["from"] = evt.From.String()
	postmap["unavailable"] = evt.Unavailable
	if !evt.LastSeen.IsZero() {
		postmap["lastSeen"] = evt.LastSeen.Unix()
	}
}

func (zc *ZPigoClient) handleChatPresenceEvent(evt *events.ChatPresence, postmap map[string]interface{}) {
	postmap["chat"] = evt.MessageSource.Chat.String()
	postmap["sender"] = evt.MessageSource.Sender.String()
	postmap["state"] = string(evt.State)
	postmap["media"] = string(evt.Media)
}

func (zc *ZPigoClient) handleLoggedOutEvent(evt *events.LoggedOut) {
	zc.SetActive(false)
	zc.UpdateSessionInfo("Status", "disconnected")
	zc.UpdateSessionInfo("Jid", "")
	zc.UpdateSessionInfo("Qrcode", "")

	if evt.Reason != 0 {
		zc.UpdateSessionInfo("LogoutReason", evt.Reason.String())
	}
}


func (zc *ZPigoClient) handleGroupInfoEvent(evt *events.GroupInfo, postmap map[string]interface{}) {
	postmap["jid"] = evt.JID.String()
	postmap["notify"] = evt.Notify
	postmap["timestamp"] = evt.Timestamp.Unix()

	if evt.Sender != nil {
		postmap["sender"] = evt.Sender.String()
	}
	if evt.Name != nil {
		postmap["name"] = evt.Name
	}
	if evt.Topic != nil {
		postmap["topic"] = evt.Topic
	}
	if len(evt.Join) > 0 {
		postmap["join"] = evt.Join
	}
	if len(evt.Leave) > 0 {
		postmap["leave"] = evt.Leave
	}
	if len(evt.Promote) > 0 {
		postmap["promote"] = evt.Promote
	}
	if len(evt.Demote) > 0 {
		postmap["demote"] = evt.Demote
	}
}

func (zc *ZPigoClient) handleJoinedGroupEvent(evt *events.JoinedGroup, postmap map[string]interface{}) {
	postmap["jid"] = evt.JID.String()
	postmap["reason"] = evt.Reason
	postmap["type"] = evt.Type
	postmap["createKey"] = evt.CreateKey

	if evt.Sender != nil {
		postmap["sender"] = evt.Sender.String()
	}
}


func (zc *ZPigoClient) handleContactEvent(evt *events.Contact, postmap map[string]interface{}) {
	postmap["jid"] = evt.JID.String()
	postmap["timestamp"] = evt.Timestamp.Unix()
	postmap["fromFullSync"] = evt.FromFullSync
	postmap["action"] = evt.Action
}

func (zc *ZPigoClient) handlePushNameEvent(evt *events.PushName, postmap map[string]interface{}) {
	postmap["jid"] = evt.JID.String()
	postmap["oldPushName"] = evt.OldPushName
	postmap["newPushName"] = evt.NewPushName

	if evt.Message != nil {
		postmap["messageId"] = evt.Message.ID
	}
}

func (zc *ZPigoClient) handleBusinessNameEvent(evt *events.BusinessName, postmap map[string]interface{}) {
	postmap["jid"] = evt.JID.String()
	postmap["oldBusinessName"] = evt.OldBusinessName
	postmap["newBusinessName"] = evt.NewBusinessName

	if evt.Message != nil {
		postmap["messageId"] = evt.Message.ID
	}
}

func (zc *ZPigoClient) handlePictureEvent(evt *events.Picture, postmap map[string]interface{}) {
	postmap["jid"] = evt.JID.String()
	postmap["author"] = evt.Author.String()
	postmap["timestamp"] = evt.Timestamp.Unix()
	postmap["remove"] = evt.Remove
	postmap["pictureId"] = evt.PictureID
}

func (zc *ZPigoClient) callWebhook(postmap map[string]interface{}) {
	webhookLogger := logger.WithComponent("Webhook").With("sessionID", zc.SessionID)

	eventTypeStr, ok := postmap["type"].(string)
	if !ok {
		webhookLogger.Error("Tipo de evento inválido no postmap")
		return
	}

	eventType := webhook.EventType(eventTypeStr)

	eventData := make(map[string]interface{})
	for k, v := range postmap {
		if k != "type" && k != "sessionId" && k != "timestamp" {
			eventData[k] = v
		}
	}

	webhookLogger.Info("Webhook preparado para envio",
		"eventType", eventType,
		"sessionID", zc.SessionID,
		"dataKeys", len(eventData))
}
