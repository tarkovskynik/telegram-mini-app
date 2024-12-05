package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func main() {
	url := "ws://localhost:8888/api/v1/store/ws"
	//url := "wss://miniapp.ultimatedivision.com/api/v1/store/ws"

	header := http.Header{}
	header.Add("Authorization", "Telegram query_id=AAHdF6IQAAAAAN0XohDhrOrc&user=%7B%22id%22%3A5060715466%2C%22first_name%22%3A%22Bob%22%2C%22last_name%22%3A%22Trader%22%2C%22username%22%3A%22defi_master%22%7D&auth_date=1677649900&hash=e2e58...")

	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	messageQueue := make(chan []byte)

	go func() {
		defer close(messageQueue)
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				return
			}

			messageQueue <- p
		}
	}()

	for message := range messageQueue {
		log.Printf("Received:\n%s\n", message)
	}
}
