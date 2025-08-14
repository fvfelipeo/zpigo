package meow

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"github.com/uptrace/bun"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"

	"zpigo/internal/db/models"
	"zpigo/internal/logger"
	"zpigo/internal/repository"
)

type SessionManager struct {
	whatsmeowClients map[string]*whatsmeow.Client
	httpClients      map[string]*resty.Client

	container *sqlstore.Container

	db          *bun.DB
	sessionRepo repository.SessionRepositoryInterface

	cacheManager *CacheManager

	mu sync.RWMutex

	logger logger.Logger

	killChannels map[string]chan bool
}

func NewSessionManager(container *sqlstore.Container, db *bun.DB, sessionRepo repository.SessionRepositoryInterface) *SessionManager {
	return &SessionManager{
		whatsmeowClients: make(map[string]*whatsmeow.Client),
		httpClients:      make(map[string]*resty.Client),
		container:        container,
		db:               db,
		sessionRepo:      sessionRepo,
		cacheManager:     GetGlobalCache(),
		logger:           NewLoggerForComponent("SessionManager"),
		killChannels:     make(map[string]chan bool),
	}
}

func (sm *SessionManager) CreateSession(sessionID string) (*whatsmeow.Client, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.logger.Info("Criando nova sessão", "sessionID", sessionID)

	if _, exists := sm.whatsmeowClients[sessionID]; exists {
		sm.logger.Warn("Tentativa de criar sessão existente", "sessionID", sessionID)
		return nil, fmt.Errorf("sessão %s já existe", sessionID)
	}

	deviceStore := sm.container.NewDevice()

	waLogger := logger.NewWhatsAppLogger("WhatsApp", DefaultLogLevel)
	client := whatsmeow.NewClient(deviceStore, waLogger)

	sm.whatsmeowClients[sessionID] = client
	sm.logger.Info("Sessão criada com sucesso", "sessionID", sessionID)

	return client, nil
}

func (sm *SessionManager) SetWhatsmeowClient(sessionID string, client *whatsmeow.Client) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.whatsmeowClients[sessionID] = client
}

func (sm *SessionManager) GetWhatsmeowClient(sessionID string) *whatsmeow.Client {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.whatsmeowClients[sessionID]
}

func (sm *SessionManager) DeleteWhatsmeowClient(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.whatsmeowClients, sessionID)
}

func (sm *SessionManager) SetHTTPClient(sessionID string, client *resty.Client) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.httpClients[sessionID] = client
}

func (sm *SessionManager) GetHTTPClient(sessionID string) *resty.Client {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.httpClients[sessionID]
}

func (sm *SessionManager) DeleteHTTPClient(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.httpClients, sessionID)
}

func (sm *SessionManager) GetSession(sessionID string) (*whatsmeow.Client, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	client, exists := sm.whatsmeowClients[sessionID]
	return client, exists
}

func (sm *SessionManager) DeleteSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	client, exists := sm.whatsmeowClients[sessionID]
	if !exists {
		return fmt.Errorf("sessão %s não encontrada", sessionID)
	}

	if client.IsConnected() {
		client.Disconnect()
	}

	delete(sm.whatsmeowClients, sessionID)

	return nil
}

func (sm *SessionManager) ConnectSession(sessionID string) error {
	client, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("sessão %s não encontrada", sessionID)
	}

	if client.IsConnected() {
		return fmt.Errorf("sessão %s já está conectada", sessionID)
	}

	if client.Store.ID == nil {
		qrChan, err := client.GetQRChannel(context.Background())
		if err != nil {
			if !errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
				return fmt.Errorf("erro ao obter canal QR: %v", err)
			}
		} else {
			err = client.Connect()
			if err != nil {
				return fmt.Errorf("erro ao conectar: %v", err)
			}

			go sm.handleQREvents(sessionID, qrChan)
			return nil
		}
	}

	return client.Connect()
}

func (sm *SessionManager) handleQREvents(sessionID string, qrChan <-chan whatsmeow.QRChannelItem) {
	logger := sm.logger.With("sessionID", sessionID).With("component", "QRHandler")

	var wasSuccessful bool

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			logger.Info("QR code gerado", "code", evt.Code)

			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			fmt.Println("QR code:", evt.Code)

			qrImage, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
			if err != nil {
				logger.Error("Erro ao gerar imagem QR", "error", err)
				continue
			}

			base64QRCode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrImage)

			err = sm.sessionRepo.UpdateQRCode(context.Background(), sessionID, base64QRCode)
			if err != nil {
				logger.Error("Erro ao salvar QR code no banco", "error", err)
			} else {
				logger.Info("QR code salvo no banco com sucesso")
			}

		case "timeout":
			logger.Warn("QR code expirou")

			err := sm.sessionRepo.SetDisconnected(context.Background(), sessionID)
			if err != nil {
				logger.Error("Erro ao atualizar sessão após timeout", "error", err)
			} else {
				logger.Info("Status da sessão voltou para disconnected após timeout do QR code", "sessionID", sessionID)
			}

			if client, exists := sm.GetSession(sessionID); exists {
				if client.IsConnected() {
					client.Disconnect()
					logger.Info("Cliente WhatsApp desconectado após timeout", "sessionID", sessionID)
				}
			}

			return

		case "success":
			logger.Info("QR code autenticado com sucesso!")
			wasSuccessful = true

			// Obter o deviceJid do cliente
			client, exists := sm.GetSession(sessionID)
			deviceJid := ""
			phone := ""
			if exists && client.Store.ID != nil {
				deviceJid = client.Store.ID.String()
				// Extrair phone number do JID (parte antes do :)
				if client.Store.ID.User != "" {
					phone = strings.Split(client.Store.ID.User, ":")[0]
				}
			}

			err := sm.sessionRepo.SetConnected(context.Background(), sessionID, phone, deviceJid)
			if err != nil {
				logger.Error("Erro ao atualizar status da sessão", "error", err)
			} else {
				logger.Info("Sessão marcada como conectada após autenticação bem-sucedida", "sessionID", sessionID, "phone", phone, "deviceJid", deviceJid)
			}

			// Limpar QR code após sucesso
			err = sm.sessionRepo.UpdateQRCode(context.Background(), sessionID, "")
			if err != nil {
				logger.Error("Erro ao limpar QR code", "error", err)
			} else {
				logger.Info("QR code limpo após autenticação bem-sucedida", "sessionID", sessionID)
			}

		default:
			logger.Info("Evento QR recebido", "event", evt.Event)
		}
	}

	// Canal QR fechado - verificar se foi por sucesso ou erro
	if wasSuccessful {
		logger.Info("Canal QR fechado após autenticação bem-sucedida", "sessionID", sessionID)
		// Não fazer nada - a sessão já foi marcada como conectada
		return
	}

	// Canal fechado sem sucesso - provavelmente erro ou cancelamento
	logger.Warn("Canal QR fechado sem sucesso", "sessionID", sessionID)

	err := sm.sessionRepo.SetDisconnected(context.Background(), sessionID)
	if err != nil {
		logger.Error("Erro ao atualizar sessão após fechamento do canal QR", "error", err)
	} else {
		logger.Info("Status da sessão voltou para disconnected após fechamento do canal QR", "sessionID", sessionID)
	}

	if client, exists := sm.GetSession(sessionID); exists {
		if client.IsConnected() {
			client.Disconnect()
			logger.Info("Cliente WhatsApp desconectado após fechamento do canal QR", "sessionID", sessionID)
		}
	}
}

func (sm *SessionManager) DisconnectSession(sessionID string) error {
	client, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("sessão %s não encontrada", sessionID)
	}

	if !client.IsConnected() {
		return fmt.Errorf("sessão %s não está conectada", sessionID)
	}

	client.Disconnect()
	return nil
}

func (sm *SessionManager) LogoutSession(sessionID string) error {
	client, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("sessão %s não encontrada", sessionID)
	}

	if !client.IsLoggedIn() {
		return fmt.Errorf("sessão %s não está logada", sessionID)
	}

	return client.Logout(context.Background())
}

func (sm *SessionManager) GenerateQRCode(sessionID string) (string, error) {
	client, exists := sm.GetSession(sessionID)
	if !exists {
		return "", fmt.Errorf("sessão %s não encontrada", sessionID)
	}

	if client.IsLoggedIn() {
		return "", fmt.Errorf("sessão %s já está autenticada", sessionID)
	}

	session, err := sm.sessionRepo.GetByID(context.Background(), sessionID)
	if err != nil {
		return "", fmt.Errorf("erro ao buscar QR code: %v", err)
	}

	if session.QRCode == "" {
		return "", fmt.Errorf("QR code não disponível. Certifique-se de que a sessão está conectada")
	}

	return session.QRCode, nil
}

// ConnectOnStartup reconecta automaticamente todas as sessões que estavam conectadas
func (sm *SessionManager) ConnectOnStartup() error {
	sm.logger.Info("🔄 Iniciando reconexão automática de sessões conectadas")

	// Buscar todas as sessões conectadas no banco
	sessions, err := sm.sessionRepo.GetAll(context.Background())
	if err != nil {
		sm.logger.Error("Erro ao buscar sessões para reconexão", "error", err)
		return err
	}

	connectedCount := 0
	for _, session := range sessions {
		if session.Status == models.StatusConnected && session.DeviceJid != "" {
			connectedCount++
			sm.logger.Info("📱 Tentando reconectar sessão",
				"sessionID", session.ID,
				"name", session.Name,
				"deviceJid", session.DeviceJid)

			// Reconectar sessão em goroutine separada
			go func(sess models.Session) {
				err := sm.reconnectSession(sess.ID, sess.DeviceJid)
				if err != nil {
					sm.logger.Error("❌ Erro ao reconectar sessão",
						"sessionID", sess.ID,
						"name", sess.Name,
						"error", err)

					// Marcar como disconnected se falhou
					sm.sessionRepo.UpdateStatus(context.Background(), sess.ID, models.StatusDisconnected)
				} else {
					sm.logger.Info("✅ Sessão reconectada com sucesso",
						"sessionID", sess.ID,
						"name", sess.Name)
				}
			}(session)
		}
	}

	if connectedCount > 0 {
		sm.logger.Info("🚀 Iniciando reconexão de sessões", "totalSessions", connectedCount)
	} else {
		sm.logger.Info("📭 Nenhuma sessão conectada encontrada para reconexão")
	}

	return nil
}

// reconnectSession reconecta uma sessão específica usando o deviceJid
func (sm *SessionManager) reconnectSession(sessionID, deviceJid string) error {
	sm.logger.Info("🔄 Iniciando reconexão da sessão", "sessionID", sessionID, "deviceJid", deviceJid)

	// Verificar se a sessão já existe
	if _, exists := sm.GetSession(sessionID); exists {
		sm.logger.Warn("Sessão já existe, pulando reconexão", "sessionID", sessionID)
		return nil
	}

	// Parse do JID para validação
	jid, err := types.ParseJID(deviceJid)
	if err != nil {
		sm.logger.Error("Erro ao fazer parse do deviceJid", "sessionID", sessionID, "deviceJid", deviceJid, "error", err)
		// Marcar como disconnected se JID inválido
		sm.sessionRepo.UpdateStatus(context.Background(), sessionID, models.StatusDisconnected)
		return fmt.Errorf("erro ao fazer parse do deviceJid: %w", err)
	}

	// Verificar se o device store existe
	deviceStore, err := sm.container.GetDevice(context.Background(), jid)
	if err != nil || deviceStore == nil {
		sm.logger.Warn("Device não encontrado no banco, sessão foi removida do WhatsApp", "sessionID", sessionID, "deviceJid", deviceJid, "error", err)
		// Marcar como disconnected se device não existe
		sm.sessionRepo.SetDisconnected(context.Background(), sessionID)
		return fmt.Errorf("device não encontrado: %w", err)
	}

	// Verificar se o device tem dados válidos
	if deviceStore == nil || deviceStore.ID == nil {
		sm.logger.Warn("Device store inválido ou sem ID, sessão precisa ser reconectada manualmente", "sessionID", sessionID)
		// Marcar como disconnected e deixar o usuário reconectar manualmente
		sm.sessionRepo.UpdateStatus(context.Background(), sessionID, models.StatusDisconnected)
		return fmt.Errorf("device store inválido ou sem ID válido")
	}

	// Usar o fluxo normal de criação de cliente (similar ao CreateSession)
	waLogger := logger.NewWhatsAppLogger("WhatsApp", DefaultLogLevel)
	client := whatsmeow.NewClient(deviceStore, waLogger)

	// Tentar conectar
	err = client.Connect()
	if err != nil {
		sm.logger.Error("Erro ao conectar cliente na reconexão", "sessionID", sessionID, "deviceJid", deviceJid, "error", err)
		// Marcar como disconnected se falhou na conexão
		sm.sessionRepo.UpdateStatus(context.Background(), sessionID, models.StatusDisconnected)
		return fmt.Errorf("erro ao conectar cliente: %w", err)
	}

	// Armazenar cliente apenas se conectou com sucesso
	sm.mu.Lock()
	sm.whatsmeowClients[sessionID] = client
	sm.mu.Unlock()

	sm.logger.Info("✅ Sessão reconectada com sucesso", "sessionID", sessionID, "deviceJid", deviceJid)
	return nil
}

func (sm *SessionManager) PairPhone(sessionID, phoneNumber string) (string, error) {
	client, exists := sm.GetSession(sessionID)
	if !exists {
		return "", fmt.Errorf("sessão %s não encontrada", sessionID)
	}

	if client.IsLoggedIn() {
		return "", fmt.Errorf("sessão %s já está autenticada", sessionID)
	}

	if !client.IsConnected() {
		if err := client.Connect(); err != nil {
			return "", fmt.Errorf("erro ao conectar: %v", err)
		}
	}

	linkingCode, err := client.PairPhone(context.Background(), phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return "", fmt.Errorf("erro ao emparelhar telefone: %v", err)
	}

	return linkingCode, nil
}

func (sm *SessionManager) GetSessionStatus(sessionID string) (bool, bool, error) {
	client, exists := sm.GetSession(sessionID)
	if !exists {
		return false, false, fmt.Errorf("sessão %s não encontrada", sessionID)
	}

	return client.IsConnected(), client.IsLoggedIn(), nil
}

func (sm *SessionManager) SetProxy(sessionID string, proxyConfig *models.Session) error {
	client, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("sessão %s não encontrada", sessionID)
	}

	if client.IsConnected() {
		return fmt.Errorf("não é possível configurar proxy com sessão conectada")
	}

	return nil
}

func (sm *SessionManager) ListSessions() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]string, 0, len(sm.whatsmeowClients))
	for sessionID := range sm.whatsmeowClients {
		sessions = append(sessions, sessionID)
	}

	return sessions
}

func (sm *SessionManager) AddEventHandler(sessionID string, handler func(any)) error {
	client, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("sessão %s não encontrada", sessionID)
	}

	client.AddEventHandler(handler)
	return nil
}

func (sm *SessionManager) GetSessionByAPIKey(apiKey, sessionID string) (*SessionInfo, error) {
	cacheKey := BuildCacheKey(apiKey, sessionID)

	sessionInfo, found := sm.cacheManager.GetSessionInfo(cacheKey)
	if !found {
		return nil, fmt.Errorf("sessão não encontrada: %s", sessionID)
	}

	return sessionInfo, nil
}

func (sm *SessionManager) ListSessionsByAPIKey(apiKey string) ([]*SessionInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var sessions []*SessionInfo

	for sessionID := range sm.whatsmeowClients {
		cacheKey := BuildCacheKey(apiKey, sessionID)
		if sessionInfo, found := sm.cacheManager.GetSessionInfo(cacheKey); found {
			sessions = append(sessions, sessionInfo)
		}
	}

	return sessions, nil
}

func (sm *SessionManager) ConnectSessionByAPIKey(apiKey, sessionID string) error {
	client, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("sessão não encontrada: %s", sessionID)
	}

	cacheKey := BuildCacheKey(apiKey, sessionID)
	if _, found := sm.cacheManager.GetSessionInfo(cacheKey); !found {
		return fmt.Errorf("sessão não autorizada: %s", sessionID)
	}

	if err := client.Connect(); err != nil {
		sm.logger.Error("Erro ao conectar sessão", "sessionID", sessionID, "error", err)
		return fmt.Errorf("erro ao conectar: %v", err)
	}

	sm.cacheManager.UpdateSessionInfo(cacheKey, "Status", "connecting")

	sm.logger.Info("Sessão conectada com sucesso", "sessionID", sessionID)
	return nil
}

func (sm *SessionManager) LogoutSessionByAPIKey(apiKey, sessionID string) error {
	client, exists := sm.GetSession(sessionID)
	if !exists {
		return fmt.Errorf("sessão não encontrada: %s", sessionID)
	}

	cacheKey := BuildCacheKey(apiKey, sessionID)
	if _, found := sm.cacheManager.GetSessionInfo(cacheKey); !found {
		return fmt.Errorf("sessão não autorizada: %s", sessionID)
	}

	if err := client.Logout(context.Background()); err != nil {
		sm.logger.Error("Erro ao fazer logout", "sessionID", sessionID, "error", err)
		return fmt.Errorf("erro ao fazer logout: %v", err)
	}

	sm.cacheManager.UpdateSessionInfo(cacheKey, "Status", "disconnected")
	sm.cacheManager.UpdateSessionInfo(cacheKey, "JID", "")
	sm.cacheManager.UpdateSessionInfo(cacheKey, "QRCode", "")

	sm.logger.Info("Logout realizado com sucesso", "sessionID", sessionID)
	return nil
}

func (sm *SessionManager) GetQRCodeByAPIKey(apiKey, sessionID string) (string, error) {
	cacheKey := BuildCacheKey(apiKey, sessionID)

	sessionInfo, found := sm.cacheManager.GetSessionInfo(cacheKey)
	if !found {
		return "", fmt.Errorf("sessão não encontrada: %s", sessionID)
	}

	if sessionInfo.QRCode == "" {
		return "", fmt.Errorf("QR Code não disponível para a sessão: %s", sessionID)
	}

	return sessionInfo.QRCode, nil
}

func (sm *SessionManager) DeleteSessionByAPIKey(apiKey, sessionID string) error {
	cacheKey := BuildCacheKey(apiKey, sessionID)
	if _, found := sm.cacheManager.GetSessionInfo(cacheKey); !found {
		return fmt.Errorf("sessão não autorizada: %s", sessionID)
	}

	if err := sm.DeleteSession(sessionID); err != nil {
		return err
	}

	sm.cacheManager.DeleteSessionInfo(cacheKey)

	return nil
}
