package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jibitesh/request-response-manager/internal/logger"
	"github.com/jibitesh/request-response-manager/internal/store"
)

type ConnectionManager struct {
	sessionService *store.SessionService
	connections    map[string]*websocket.Conn
	connMu         sync.RWMutex
	upgrader       websocket.Upgrader
	pingFreq       time.Duration
	closeOnce      sync.Once
}

func NewConnectionManager(sessionService *store.SessionService) *ConnectionManager {
	return &ConnectionManager{
		sessionService: sessionService,
		connections:    make(map[string]*websocket.Conn),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		pingFreq: 10 * time.Second,
	}
}

func (cm *ConnectionManager) HandleWSClient(w http.ResponseWriter, r *http.Request) {
	conn, err := cm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("upgrade: %v", err)
		http.Error(w, "Upgrade failed!", http.StatusBadRequest)
		return
	}

	sessionId := uuid.NewString()
	cm.connMu.Lock()
	cm.connections[sessionId] = conn
	cm.connMu.Unlock()

	if _, err := cm.sessionService.AddSession(r.Context(), sessionId); err != nil {
		logger.Error("set session: %v", err)
		conn.Close()
		cm.removeConnection(sessionId)
		return
	}
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("Connection closed normally for sessionId: %s. Error: %v", sessionId, err)
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("Connection closed abnormally for sessionId: %s. Error: %v", sessionId, err)
			} else {
				logger.Error("Failed to read message from sessionId: %s. Error: %v", sessionId, err)
			}
			if err := cm.sessionService.RemoveSession(r.Context(), sessionId); err != nil {
				logger.Error("Failed to remove session: %s with error: %v", sessionId, err)
				cm.removeConnection(sessionId)
			}
			break
		}
		if messageType == websocket.TextMessage {
			logger.Info("Received message from sessionId: %s. Message: %v", sessionId, message)
		}
	}
}

func (cm *ConnectionManager) HandleWSSend(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[1] != "ws" || parts[2] != "send" {
		http.Error(w, "Invalid websocket path format", http.StatusBadRequest)
		return
	}
	sessionId := parts[3]
	if sessionId == "" {
		logger.Info("Received empty session id.")
		http.Error(w, "Missing session id", http.StatusBadRequest)
		return
	}

	conn, err := cm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade sessionId: %s to websocket. Error: %v", sessionId, err)
		http.Error(w, "Failed to upgrade session to websocket.", http.StatusBadRequest)
		return
	}
	defer conn.Close()
	logger.Info("Upgraded sessionId: %s to websocket.", sessionId)

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorf("Connection closed normally for sessionId: %s. Error: %v", sessionId, err)
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorf("Connection closed abnormally for sessionId: %s. Error: %v", sessionId, err)
			} else {
				logger.Errorf("Failed to read message from sessionId: %s. Error: %v", sessionId, err)
			}
			break
		}
		logger.Info("Received message from sessionId: %s. Message: %s", sessionId, string(message))

		if messageType == websocket.TextMessage {
			cm.connMu.RLock()
			conn, ok := cm.connections[sessionId]
			cm.connMu.RUnlock()
			if !ok || conn == nil {
				http.Error(w, "WebSocket redirect: Session not connected to this instance.", http.StatusGone)
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				http.Error(w,
					fmt.Sprintf("Failed to redirect websocket message to websocket of sessionid: %s.", sessionId),
					http.StatusInternalServerError,
				)
				return
			}

			_ = cm.sessionService.RefreshSession(r.Context(), sessionId)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
			continue
		}
	}
	return
}

// Handles POST /send {session_id, message}
func (cm *ConnectionManager) HandleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SessionId string `json:"sessionId"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Ownership check
	si, err := cm.sessionService.GetSession(r.Context(), req.SessionId)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if si.Instance.Name != cm.sessionService.Instance().Name {
		http.Error(w, "Session owned by different instance.", http.StatusBadRequest)
		return
	}

	// Send message to local connection
	cm.connMu.RLock()
	conn, ok := cm.connections[req.SessionId]
	cm.connMu.RUnlock()
	if !ok || conn == nil {
		http.Error(w, "Session not connected to this instance.", http.StatusGone)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte(req.Message)); err != nil {
		http.Error(w, "Failed to write to websocket of sessionid.", http.StatusInternalServerError)
		return
	}

	_ = cm.sessionService.RefreshSession(r.Context(), req.SessionId)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (cm *ConnectionManager) readLoop(sessionId string, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		cm.removeConnection(sessionId)
		_ = cm.sessionService.RemoveSession(context.Background(), sessionId)
		log.Println("Session s closed and Connection removed.", sessionId)
	}()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			return
		}
		_ = cm.sessionService.RefreshSession(context.Background(), sessionId)
		err = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	}
}

func (cm *ConnectionManager) pinger(sessionId string, conn *websocket.Conn) {
	ticker := time.NewTicker(cm.pingFreq)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
			return
		}
	}
}

func (cm *ConnectionManager) removeConnection(sessionId string) {
	cm.connMu.Lock()
	defer cm.connMu.Unlock()
	delete(cm.connections, sessionId)
}

func (cm *ConnectionManager) CloseAllConnections() error {
	var firstError error
	cm.closeOnce.Do(func() {
		cm.connMu.Lock()
		defer cm.connMu.Unlock()
		for sid, conn := range cm.connections {
			if err := conn.Close(); err != nil && firstError == nil {
				firstError = err
			}
			_ = cm.sessionService.RemoveSession(context.Background(), sid)
		}
		cm.connections = make(map[string]*websocket.Conn)
	})
	return firstError
}
