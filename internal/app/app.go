package app

import (
	"log/slog"
	"signalling/internal/app/websocket"
	ssogrpc "signalling/internal/clients/sso/grpc"
	"signalling/internal/config"
)

type App struct {
	WebSocketServer *websocket.App
}

func New(
	log *slog.Logger,
	cfg *config.Config,
	sso *ssogrpc.Client,
) *App {

	ws := websocket.New(log, cfg.Port, sso)

	return &App{
		WebSocketServer: ws,
	}
}
