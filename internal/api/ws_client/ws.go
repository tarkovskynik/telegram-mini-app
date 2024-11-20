package main

import (
	"log"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
)

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

func main() {
	url := "ws://localhost:8888/api/v1/ws/5060715466"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	messageQueue := make(chan Message)

	go func() {
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				return
			}

			log.Printf("Received:\n%s\n", p)
		}
	}()

	go func() {
		for message := range messageQueue {
			mJson, err := json.MarshalIndent(message, "", "  ")
			if err != nil {
				log.Println("json marshal error:", err)
			}
			err = conn.WriteMessage(websocket.TextMessage, mJson)
			if err != nil {
				log.Println("write error:", err)
				return
			}
			log.Printf("Sent:\n%s\n", string(mJson))
		}
	}()

	appFirstStart := false
	gameStarted := false
	hitCount := 0
	ticker := time.NewTicker(5 * time.Second)

	for range ticker.C {
		if !appFirstStart {
			appFirstStart = true
			messageQueue <- Message{Type: "player_state"}

		} else if !gameStarted {
			messageQueue <- Message{Type: "game_start"}
			gameStarted = true

		} else {
			if hitCount < 5 {
				messageQueue <- Message{Type: "ball_hit"}
				hitCount++
			} else {
				messageQueue <- Message{Type: "ball_dropped"}
				gameStarted = false
				hitCount = 0

				time.Sleep(1 * time.Second)
			}
		}
	}
}
