package server

import (
	"context"
	"net/http"
	"strconv"
	"sync"

	"github.com/jibitesh/request-response-manager/internal/config"
	"github.com/jibitesh/request-response-manager/internal/logger"
	"github.com/jibitesh/request-response-manager/internal/store"
	"github.com/jibitesh/request-response-manager/internal/ws"
	"github.com/jibitesh/request-response-manager/pkg/instance"
)

type Server struct {
	cfg            *config.Config
	httpSrv        *http.Server
	sessionService *store.SessionService
	wsManager      *ws.ConnectionManager
	mu             sync.Mutex
}

func NewServer(cfg *config.Config, instance *instance.Instance) (*Server, error) {
	sessionStore, err := store.NewRedisStore(cfg, instance)
	if err != nil {
		panic(err)
	}
	sessionService := store.NewSessionService(instance, sessionStore)
	wsManager := ws.NewConnectionManager(sessionService)

	mux := http.NewServeMux()
	logger.Info("setting /ws as client websocket handler")
	mux.HandleFunc("/ws", wsManager.HandleWSClient)
	logger.Info("setting /ws/send/{id} as micro-service session send handler")
	mux.HandleFunc("/ws/send/", wsManager.HandleWSSend)
	logger.Info("setting /session/{id} as session lookup handler")
	mux.HandleFunc("/session/", ws.SessionLookupHandler(sessionService))
	logger.Info("setting /send as REST session send handler")
	mux.HandleFunc("/send", wsManager.HandleSend)

	httpSrv := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.Server.Port),
		Handler: mux,
	}

	return &Server{
		cfg:            cfg,
		httpSrv:        httpSrv,
		sessionService: sessionService,
		wsManager:      wsManager,
	}, nil
}

func (s *Server) Start() error {
	logger.Info("starting server on port %d", s.cfg.Server.Port)
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return err
	}
	if err := s.wsManager.CloseAllConnections(); err != nil {
		logger.Info("warning: ws manager close error: %v", err)
	}
	return nil
}
