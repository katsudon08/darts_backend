package main

import (
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

type Handler func (*websocket.Conn, string)

var clients = make(map[string]map[*websocket.Conn]bool)
var broadcast = make(chan Data)

func handleHello(w http.ResponseWriter, r *http.Request) {
	hello := []byte("hello world")
	_, err := w.Write(hello)

	if err != nil {
		log.Print(err)
	}
}

func handleConnection(ws *websocket.Conn) {
	defer ws.Close()

	key := ws.Request().URL.String()[1:]
	clients[key] = make(map[*websocket.Conn]bool)
	clients[key][ws] = true

	fmt.Println(clients)

	// メッセージの受信
	for {
		msg := ""

		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			if err.Error() == "EOF" {
				log.Fatal(fmt.Errorf("read %s", err))
			}
			log.Print(err)
		}

		fmt.Println(msg)

		data := Data{key, msg}

		fmt.Println(data)

		// 受取ったメッセージをbroadcastチャネルに送る(awaitのような感じ)
		broadcast <- data
	}
}

func handleMessage() {
	for {
		// broadcastからメッセージを受取る
		data := <- broadcast

		// 各クライアントへのメッセージの送信
		for client := range clients[data.Key] {
			err := websocket.Message.Send(client, data.Msg)
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