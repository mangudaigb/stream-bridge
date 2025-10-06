package store

import (
	"context"
	"log"
	"time"

	"github.com/jibitesh/request-response-manager/pkg/instance"
)

//var ErrNotFound = errors.New("session not found")

type SessionInfo struct {
	SessionId string             `json:"session_id"`
	Instance  *instance.Instance `json:"instance"`
	CreatedAt time.Time          `json:"created_at"`
}

type SessionService struct {
	instance     *instance.Instance
	sessionStore SessionStore
}

func NewSessionService(instance *instance.Instance, store SessionStore) *SessionService {
	return &SessionService{
		instance:     instance,
		sessionStore: store,
	}
}

func (ss *SessionService) AddSession(ctx context.Context, sessionId string) (bool, error) {
	si := &SessionInfo{
		SessionId: sessionId,
		Instance:  ss.instance,
		CreatedAt: time.Now(),
	}
	if err := ss.sessionStore.Set(ctx, sessionId, si); err != nil {
		log.Printf("Error saving session: %v", err)
		return false, err
	}
	return true, nil
}

func (ss *SessionService) GetSession(ctx context.Context, sessionId string) (*SessionInfo, error) {
	return ss.sessionStore.Get(ctx, sessionId)
}

func (ss *SessionService) RemoveSession(ctx context.Context, sessionId string) error {
	return ss.sessionStore.Delete(ctx, sessionId)
}

func (ss *SessionService) RefreshSession(ctx context.Context, sessionId string) error {
	return ss.sessionStore.Refresh(ctx, sessionId)
}

func (ss *SessionService) Instance() *instance.Instance {
	return ss.instance
}
