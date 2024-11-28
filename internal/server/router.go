package server

import (
	"github.com/gorilla/websocket"
	"sync"
)

type Connection struct {
	Conn   *websocket.Conn
	UserID int64
}

type Hub struct {
	connections map[int64]*Connection //Map userID -> Connection
	mu          sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[int64]*Connection),
	}
}

func (h *Hub) AddConnection(userID int64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[userID] = &Connection{
		Conn:   conn,
		UserID: userID,
	}
}

func (h *Hub) RemoveConnection(userID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.connections, userID)
}

func (h *Hub) GetConnection(userID int64) (*Connection, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conn, exists := h.connections[userID]
	return conn, exists
}
