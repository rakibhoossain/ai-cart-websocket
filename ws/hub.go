package ws

import (
	"log"
	"sync"
)

type ChannelMessage struct {
	Channel string
	Data    []byte
}

type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Map channel name to subscribed clients (e.g. "shop:uuid", "user:uuid")
	channelsMu sync.RWMutex
	channels   map[string]map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Inbound messages for a named channel.
	sendToChannel chan ChannelMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:     make(chan []byte),
		sendToChannel: make(chan ChannelMessage, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		clients:       make(map[*Client]bool),
		channels:      make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("Client registered for channel: %s", client.channel)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// Remove from all channel subscriptions
				h.channelsMu.Lock()
				for ch, subs := range h.channels {
					delete(subs, client)
					if len(subs) == 0 {
						delete(h.channels, ch)
					}
				}
				h.channelsMu.Unlock()
				log.Printf("Client unregistered from channel: %s", client.channel)
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

		case chMsg := <-h.sendToChannel:
			h.channelsMu.RLock()
			clients := h.channels[chMsg.Channel]
			h.channelsMu.RUnlock()
			for client := range clients {
				select {
				case client.send <- chMsg.Data:
				default:
					close(client.send)
					delete(h.clients, client)
					h.channelsMu.Lock()
					delete(h.channels[chMsg.Channel], client)
					h.channelsMu.Unlock()
				}
			}
		}
	}
}

func (h *Hub) Broadcast(data []byte) {
	h.broadcast <- data
}

// SendToChannel delivers data to all clients subscribed to the named channel.
func (h *Hub) SendToChannel(channel string, data []byte) {
	h.sendToChannel <- ChannelMessage{Channel: channel, Data: data}
}

// SubscribeToChannel adds a client to a named channel.
func (h *Hub) SubscribeToChannel(client *Client, channel string) {
	h.channelsMu.Lock()
	defer h.channelsMu.Unlock()
	if _, ok := h.channels[channel]; !ok {
		h.channels[channel] = make(map[*Client]bool)
	}
	h.channels[channel][client] = true
	log.Printf("Client subscribed to channel: %s", channel)
}

// UnsubscribeFromChannel removes a client from a named channel.
func (h *Hub) UnsubscribeFromChannel(client *Client, channel string) {
	h.channelsMu.Lock()
	defer h.channelsMu.Unlock()
	if subs, ok := h.channels[channel]; ok {
		delete(subs, client)
		if len(subs) == 0 {
			delete(h.channels, channel)
		}
		log.Printf("Client unsubscribed from channel: %s", channel)
	}
}
