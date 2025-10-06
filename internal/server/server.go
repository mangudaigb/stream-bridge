package server

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/jibitesh/request-response-manager/configs"
	"github.com/jibitesh/request-response-manager/internal/store"
	"github.com/jibitesh/request-response-manager/internal/ws"
	"github.com/jibitesh/request-response-manager/pkg/instance"
)

type Server struct {
	cfg            *configs.Config
	httpSrv        *http.Server
	sessionService *store.SessionService
	wsManager      *ws.ConnectionManager
	mu             sync.Mutex
}

func NewServer(cfg *configs.Config, instance *instance.Instance) (*Server, error) {
	sessionStore, err := store.NewRedisStore(cfg, instance)
	if err != nil {
		panic(err)
	}
	sessionService := store.NewSessionService(instance, sessionStore)
	wsManager := ws.NewConnectionManager(sessionService)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsManager.HandleWSClient)
	mux.HandleFunc("/session/", ws.SessionLookupHandler(sessionService))
	mux.HandleFunc("/send", wsManager.HandleSend)
	mux.HandleFunc("/ws/send/", wsManager.HandleWSSend)

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
	log.Printf("starting server on port %d", s.cfg.Server.Port)
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return err
	}
	if err := s.wsManager.CloseAllConnections(); err != nil {
		log.Printf("warning: ws manager close error: %v", err)
	}
	return nil
}
