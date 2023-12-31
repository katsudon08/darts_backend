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

func initClients() {
	clients[GROUP] = make(map[*websocket.Conn]bool)
	clients[TURN] = make(map[*websocket.Conn]bool)
	clients[USERS] = make(map[*websocket.Conn]bool)
}

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
	clients[key][ws] = true

	fmt.Println("clients: ", clients[key])

	// メッセージの受信
	for {
		msg := ""

		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			if err.Error() == "EOF" {
				delete(clients[key], ws)
				break
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

		fmt.Println("msg/clients: ", clients[data.Key])

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
	initClients()
	http.HandleFunc("/", handleHello)
	http.Handle(fmt.Sprintf("/%s", GROUP), websocket.Handler(handleConnection))
	http.Handle(fmt.Sprintf("/%s", TURN), websocket.Handler(handleConnection))
	http.Handle(fmt.Sprintf("/%s", USERS), websocket.Handler(handleConnection))
	go handleMessage()

	fmt.Println("serving at http://localhost:8080....")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}