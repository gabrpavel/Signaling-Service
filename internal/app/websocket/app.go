package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log/slog"
	"net"
	"net/http"
	"signalling/internal/clients/sso/grpc"
	"signalling/internal/lib/logger/sl"
	"signalling/internal/server"
	"signalling/internal/service/metrics"
)

type App struct {
	log      *slog.Logger
	upgrader websocket.Upgrader
	port     int
	sso      *grpc.Client
	hub      *server.Hub
}

func New(
	log *slog.Logger,
	port int,
	sso *grpc.Client,
) *App {
	return &App{
		log: log,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		port: port,
		sso:  sso,
		hub:  server.NewHub(),
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	const op = "websocketapp.Run"

	log := a.log.With(
		slog.String("op", op),
		slog.Int("port", a.port),
	)

	log.Info("starting metrics server")

	go func() {
		err := metrics.StartMetricsServer(":9091")
		if err != nil {
			log.Error("failed to start metrics server", sl.Err(err))
		}
	}()

	log.Info("starting WebSocket server")

	http.HandleFunc("/ws", a.authMiddleware(a.handleConnections))

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("WebSocket server is running", slog.String("addr", l.Addr().String()))

	if err := http.Serve(l, nil); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) handleConnections(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int64)
	if !ok {
		a.log.Error("userID not found in context")
		http.Error(w, "User ID missing", http.StatusInternalServerError)
		return
	}

	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		a.log.Error("connection upgrade failed", sl.Err(err))
		return
	}
	defer conn.Close()

	a.hub.AddConnection(userID, conn)
	defer a.hub.RemoveConnection(userID)

	a.log.Info("WebSocket connection established", slog.Int64("userID", userID))

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			a.log.Error("error reading message", sl.Err(err))
			break
		}
		a.handleMessages(userID, message)
	}
}

// offer - SDP offer from one user to another
// answer - SDP answer from the target user
// ice - ICE candidate to establish a P2P connection
func (a *App) handleMessages(userID int64, message []byte) {
	var msg struct {
		Type    string `json:"type"`
		Target  int64  `json:"target"`
		Payload string `json:"payload"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		a.log.Error("failed to unmarshal message", sl.Err(err))
		return
	}

	switch msg.Type {
	case "offer", "answer", "ice":
		if targetConn, ok := a.hub.GetConnection(msg.Target); ok {
			if err := targetConn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				a.log.Error("failed to send message", sl.Err(err))
			}
		} else {
			a.log.Error("target user not connected", slog.Int64("targetID", msg.Target))
		}
	default:
		a.log.Warn("unknown message type", slog.String("type", msg.Type))
	}
}

func (a *App) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Authorization token required", http.StatusUnauthorized)
			return
		}

		// Verify token using the SSO client
		isValid, userID, _, err := a.sso.VerifyToken(r.Context(), token)
		if err != nil || !isValid {
			a.log.Error("invalid token", slog.String("error", err.Error()))
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
