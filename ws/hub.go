package ws

import (
	"log"
)

type UserMessage struct {
	UserID string
	Data   []byte
}

type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Map user ID to clients
	users map[string]map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Inbound messages intended for specific users.
	sendToUser chan UserMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		sendToUser: make(chan UserMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		users:      make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			if _, ok := h.users[client.userID]; !ok {
				h.users[client.userID] = make(map[*Client]bool)
			}
			h.users[client.userID][client] = true
			log.Printf("Client registered: %s", client.userID)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				if _, ok := h.users[client.userID]; ok {
					delete(h.users[client.userID], client)
					if len(h.users[client.userID]) == 0 {
						delete(h.users, client.userID)
					}
				}
				log.Printf("Client unregistered: %s", client.userID)
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}

		case userMsg := <-h.sendToUser:
			if clients, ok := h.users[userMsg.UserID]; ok {
				for client := range clients {
					select {
					case client.send <- userMsg.Data:
					default:
						close(client.send)
						delete(h.clients, client)
						delete(clients, client)
					}
				}
			}
		}
	}
}

func (h *Hub) SendToUser(userID string, data []byte) {
	h.sendToUser <- UserMessage{UserID: userID, Data: data}
}

func (h *Hub) Broadcast(data []byte) {
	h.broadcast <- data
}
