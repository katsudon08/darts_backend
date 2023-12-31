package main

import (
	"fmt"
	"log"
	"encoding/json"
	"net/http"
	"golang.org/x/net/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan DATA)

func handleHello(w http.ResponseWriter, r *http.Request) {
	hello := []byte("hello world")
	_, err := w.Write(hello)

	if err != nil {
		log.Print(err)
	}
}

func handleConnection(ws *websocket.Conn) {
	defer ws.Close()

	clients[ws] = true

	// メッセージの受信
	for {
		var data DATA
		msg := ""

		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			log.Print(err)
		}

		err = json.Unmarshal([]byte(msg), &data)
		if err != nil {
			log.Print(err)
		}

		fmt.Println(data)

		// 受取ったメッセージをbroadcastチャネルに送る(awaitのような感じ)
		broadcast <- data
	}
}

func handleMessage() {
	for {
		// broadcastからメッセージを受取る
		data := <- broadcast

		if data.Key == "" && data.Detail == "" && data.Value == "" {
			continue
		}

		// 各クライアントへのメッセージの送信
		for client := range clients {
			err := websocket.Message.Send(client, fmt.Sprintf("{Key: %s, Detail: %s, Value: %s}", data.Key, data.Detail, data.Value))
			if err != nil {
				log.Print(err)
			}
		}
	}
}

func main() {
	http.HandleFunc("/", handleHello)
	http.Handle("/ws", websocket.Handler(handleConnection))
	go handleMessage()

	fmt.Println("serving at http://localhost:8080....")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}