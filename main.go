package main

import (
	"fmt"
	"log"
	"net/http"
	"golang.org/x/net/websocket"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	hello := []byte("hello world")
	_, err := w.Write(hello)

	if err != nil {
		log.Fatal(err)
	}
}

func webSocketHandler(w http.ResponseWriter, r *http.Request) {
	websocket.Handler(func(ws *websocket.Conn) {
		// 関数ブロックを抜ける前に必ず実行する
		defer ws.Close()

		err := websocket.Message.Send(ws, "Server: Hello World!")
		if err != nil {
			log.Fatal(err)
		}

		for {
			msg := ""
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				if err.Error() == "EOF" {
					log.Println(fmt.Errorf("read %s", err))
					break
				}
				log.Fatal(err)
			}

			err = websocket.Message.Send(ws, fmt.Sprintf("Server: \"%s\" received", msg))
			if err != nil {
				log.Fatal(err)
			}
		}
	}).ServeHTTP(w, r)
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/socket", webSocketHandler)
	fmt.Println("serving at http://localhost:8080....")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}