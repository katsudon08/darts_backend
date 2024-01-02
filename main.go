package main

import (
	"fmt"
	"log"
	"net/http"
	"golang.org/x/net/websocket"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan string)

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
		msg := ""

		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			log.Print(err)
		}

		fmt.Println(msg)

		// 受取ったメッセージをbroadcastチャネルに送る(awaitのような感じ)
		broadcast <- msg
	}
}

func handleMessage() {
	for {
		// broadcastからメッセージを受取る
		msg := <- broadcast

		// 各クライアントへのメッセージの送信
		for client := range clients {
			err := websocket.Message.Send(client, msg)
			if err != nil {
				log.Print(err)
			}
		}
	}
}

func main() {
	http.HandleFunc("/", handleHello)
	http.Handle(fmt.Sprintf("/%s", GROUP), websocket.Handler(handleConnection))
	http.Handle(fmt.Sprintf("/%s", TURN), websocket.Handler(handleConnection))
	go handleMessage()

	fmt.Println("serving at http://localhost:8080....")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}