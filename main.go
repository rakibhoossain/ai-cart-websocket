package main

import (
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

	if err := auth.LoadPublicKey(cfg.JWTPublicKey); err != nil {
		log.Fatalf("Failed to load public key: %v", err)
	}

	hub := ws.NewHub()
	go hub.Run()

	// Start RabbitMQ Consumer in a goroutine
	go mq.StartConsumer(cfg.RabbitMQURL, cfg.RabbitMQQueue, hub)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// 1. Upgrade the connection immediately
		conn, err := ws.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}

		// Helper to close connection causing policy violation
		closeWithPolicyViolation := func(reason string) {
			log.Printf("Closing connection: %s", reason)
			msg := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, reason)
			conn.WriteMessage(websocket.CloseMessage, msg)
			conn.Close()
		}

		tokenString := r.URL.Query().Get("token")
		if tokenString == "" {
			closeWithPolicyViolation("Unauthorized: missing token")
			return
		}

		// 2. Validate the token
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			closeWithPolicyViolation("Unauthorized: invalid token")
			return
		}

		// Try to find user ID from various claims
		var userID string

		// 1. Try 'sub' (Standard Subject)
		if sub, ok := claims["sub"].(string); ok {
			userID = sub
		}

		// 2. Try 'user_id' (Custom)
		if userID == "" {
			if uid, ok := claims["user_id"].(string); ok {
				userID = uid
			}
		}

		// 3. Fallback to 'email' (UPN)
		if userID == "" {
			if email, ok := claims["email"].(string); ok {
				userID = email
			} else if upn, ok := claims["upn"].(string); ok {
				userID = upn
			}
		}

		if userID == "" {
			closeWithPolicyViolation("Unauthorized: no user identifier found")
			return
		}

		// Extract entity type from groups to distinguish between customer and admin users
		// The Java backend sets "groups" claim to the entity type (e.g. "customer", "user") or roles
		entityType := "default"
		if groups, ok := claims["groups"].([]interface{}); ok && len(groups) > 0 {
			if g, ok := groups[0].(string); ok {
				entityType = g
			}
		}

		// Create a unique ID by combining entity type and ID
		// e.g. "customer:123", "user:456"
		uniqueUserID := entityType + ":" + userID

		// 3. Register client with hub
		ws.ServeClient(hub, conn, uniqueUserID)
	})

	log.Printf("Server started on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
