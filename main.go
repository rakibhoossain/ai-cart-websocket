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


	// Channel-scoped WebSocket endpoint
	// Clients connect to /ws/channel/{channel}?token=...
	http.HandleFunc("/ws/channel/{channel}", func(w http.ResponseWriter, r *http.Request) {
		channel := r.PathValue("channel")

		fmt.Printf("channel Id %s", channel)
		if channel == "" {
			log.Println("Missing channel")
			return
		}

		conn, err := ws.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}

		closeWithPolicyViolation := func(reason string) {
			log.Printf("[WS-CHANNEL] Closing connection: %s", reason)
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

		// Get user ID from sub for client identity
		var userID string
		if sub, ok := claims["sub"].(string); ok {
			userID = sub
		}
		var entityType string
		if entity, ok := claims["entityType"].(string); ok {
			entityType = entity
		}

		tokenChannel, _ := claims["channel"].(string)
		if tokenChannel == "" || tokenChannel != channel {
			closeWithPolicyViolation("Unauthorized: token not authorized for this channel")
			return
		}

		// We still require a valid user token to connect to channels
		if userID == "" || entityType == "" {
			closeWithPolicyViolation("Unauthorized: invalid user identity")
			return
		}

		uniqueUserID := entityType + ":" + userID
		log.Printf("[WS-CHANNEL] Client %s joining channel: %s", uniqueUserID, channel)

		// Register and automatically subscribe
		client := ws.ServeClient(hub, conn, uniqueUserID)
		hub.SubscribeToChannel(client, channel)
	})

	log.Printf("Server started on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
