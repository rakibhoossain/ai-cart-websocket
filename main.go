package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rakib/go-websocket-service/auth"
	"github.com/rakib/go-websocket-service/config"
	"github.com/rakib/go-websocket-service/mq"
	"github.com/rakib/go-websocket-service/ws"
)

func main() {
	cfg := config.Load()

	if err := auth.InitJWT(cfg.JWTSecret); err != nil {
		log.Fatalf("Failed to initialize JWT: %v", err)
	}

	hub := ws.NewHub()
	go hub.Run()

	// Start RabbitMQ Consumer in a goroutine (Optional)
	if cfg.RabbitMQURL != "" {
		go mq.StartConsumer(cfg.RabbitMQURL, cfg.RabbitMQQueue, hub)
	} else {
		log.Println("RabbitMQ URL not set, skipping RabbitMQ consumer start")
	}

	// Health Check API
	rootHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/health", rootHandler)

	// Gateway API for generating tokens via HTTP
	http.HandleFunc("/api/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Validate Secret Key
		secret := r.Header.Get("X-API-Secret")
		if cfg.APISecret == "" || secret != cfg.APISecret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req auth.TokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request: invalid payload", http.StatusBadRequest)
			return
		}

		res, err := auth.GenerateToken(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to generate token: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
	})

	// Gateway API for sending messages via HTTP
	http.HandleFunc("/api/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Validate Secret Key
		secret := r.Header.Get("X-API-Secret")
		if cfg.APISecret == "" || secret != cfg.APISecret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var msg mq.MqMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if msg.Channel == "" {
			http.Error(w, "Missing channel", http.StatusBadRequest)
			return
		}

		hub.SendToChannel(msg.Channel, msg.Data)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Message queued"))
	})

	// WebSocket endpoint
	// Clients connect to /ws?token=... (channel is derived directly from validated token claims)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := ws.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}

		closeWithPolicyViolation := func(reason string) {
			log.Printf("[WS] Closing connection: %s", reason)
			msg := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, reason)
			conn.WriteMessage(websocket.CloseMessage, msg)
			conn.Close()
		}

		tokenString := r.URL.Query().Get("token")
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			closeWithPolicyViolation("Unauthorized: invalid token")
			return
		}

		// Channel derived directly from token sub claim
		channel, ok := claims["sub"].(string)
		if !ok || channel == "" {
			closeWithPolicyViolation("Unauthorized: missing channel in token sub")
			return
		}

		log.Printf("[WS] Client connected to channel: %s", channel)

		// Register and automatically subscribe
		client := ws.ServeClient(hub, conn, channel)
		hub.SubscribeToChannel(client, channel)
	})

	log.Printf("Server started on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
