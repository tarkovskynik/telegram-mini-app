package main

import (
	"log"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
)

func main() {
	url := "ws://localhost:8888/api/v1/ws/5060715466"
	//url := "wss://miniapp.ultimatedivision.com/api/v1/ws/5060715466"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	type Message struct {
		Type string      `json:"type"`
		Data interface{} `json:"data,omitempty"`
	}

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
	energyCount := 10
	hitCount := 0
	cyclesToResetEnergy := 1
	ticker := time.NewTicker(2 * time.Second)

	for range ticker.C {
		if !appFirstStart {
			appFirstStart = true
			messageQueue <- Message{Type: "player_state"}

		} else if !gameStarted {
			if cyclesToResetEnergy <= 0 {
				//messageQueue <- Message{Type: "energy_recharge"}
				cyclesToResetEnergy = 1
				energyCount = 3
				continue
			}

			if energyCount <= 0 {
				cyclesToResetEnergy--
			}

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
				energyCount--

				time.Sleep(1 * time.Second)
			}
		}
	}
}
