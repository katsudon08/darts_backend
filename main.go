package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/websocket"
)

type teamcodeJSON struct {
	Teamcode string `json:"teamcode"`
}

var users = make(map[*websocket.Conn]string)
var usersMsg = ""
var clients = make(map[string]map[*websocket.Conn]bool)
var broadcast = make(chan Data)
var listeners = make(map[string]net.Listener)

func initClients() {
	clients[TURN] = make(map[*websocket.Conn]bool)
	clients[USERS] = make(map[*websocket.Conn]bool)
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	hello := []byte("hello world")
	_, err := w.Write(hello)

	if err != nil {
		panic(err)
	}
}

// チームコードのコネクション

func handleCreateTeamCode(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body := r.Body
		defer body.Close()

		buf := new(bytes.Buffer)
		io.Copy(buf, body)

		var data teamcodeJSON
		json.Unmarshal(buf.Bytes(), &data)

		teamcode := data.Teamcode
		fmt.Println("teamcode_create:", teamcode)
		fmt.Println(r.Host)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", fmt.Sprintf("http://%s, https://%s", r.Host, r.Host))
		w.Header().Set("Access-Control-Allow-Methods", "POST")

		for key := range listeners {
			if key == teamcode {
				return
			}
		}

		// TODO: チームコードのwebsocketを作成
		listener, ch := server(":8080", teamcode)
		fmt.Println("websocket: ", listener.Addr())

		listeners[teamcode] = listener

		_, err := w.Write([]byte(teamcode))
		if err != nil {
			panic(err)
		}

		fmt.Println(ch)
	}
}

func handleJoinTeamCode(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body := r.Body
		defer body.Close()

		buf := new(bytes.Buffer)
		io.Copy(buf, body)

		var data teamcodeJSON
		json.Unmarshal(buf.Bytes(), &data)

		teamcode := data.Teamcode
		fmt.Println("teamcode_join:", teamcode)
		fmt.Println(r.Host)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", fmt.Sprintf("http://%s, https://%s", r.Host, r.Host))
		w.Header().Set("Access-Control-Allow-Methods", "POST")

		for key := range listeners {
			if key == teamcode {
				_, err := w.Write([]byte(teamcode))
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

// ターンのコネクション

func handleTurnConnection(ws *websocket.Conn) {

}

// ユーザーのコネクション

func handleUsersConnection(ws *websocket.Conn) {

}

func handleConnection(ws *websocket.Conn) {
	defer ws.Close()

	key := ws.Request().URL.String()[1:]
	clients[key][ws] = true

	fmt.Println("clients:", clients[key])

	// メッセージの受信
	for {
		msg := ""

		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			if err.Error() == "EOF" {
				delete(clients[key], ws)
				break
			}
			panic(err)
		}

		fmt.Println(msg)

		for key, value := range listeners {
			if key == msg {
				value.Close()
			}
		}

		data := Data{key, msg}

		// if key == TEAM_CODE {
		// 	if msg == "" {
		// 		data = Data{key, teamcode}
		// 	} else {
		// 		teamcode = msg
		// 	}
		// }

		fmt.Println(data)

		// 受取ったメッセージをbroadcastチャネルに送る(awaitのような感じ)
		broadcast <- data
	}
}

func handleUsersConnections(ws *websocket.Conn) {
	defer ws.Close()

	clients[USERS][ws] = false

	fmt.Println("clients:", clients[USERS])

	for {
		usersMsg = ""
		user := ""

		err := websocket.Message.Receive(ws, &user)
		if err != nil {
			if err.Error() == "EOF" {
				delete(clients[USERS], ws)
				break
			}
			panic(err)
		}

		fmt.Println("preUser:",users[ws], "newUser:", user)

		// ユーザーのグループ変更
		if user != users[ws] {
			users[ws] = user
		}

		// グループの人数制限
		if len(users) < NUMBER_OF_TEAM && !clients[USERS][ws] {
			clients[USERS][ws] = true
		}

		fmt.Println(users)

		for _, user := range users {
			usersMsg += fmt.Sprintf("%s ", user)
		}

		fmt.Println(usersMsg)

		data := Data{USERS, usersMsg}

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
				panic(err)
			}
		}
	}
}

func routes(teamcode string) (mux *http.ServeMux) {
	mux = http.NewServeMux()
	mux.Handle(fmt.Sprintf("/%s/%s", TURN, teamcode), websocket.Handler(handleConnection))
	return
}

func server(addr string, teamcode string) (listener net.Listener, ch chan error) {
	ch = make(chan error)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	go func()  {
		mux := routes(teamcode)
		ch <- http.Serve(listener, mux)
	}()

	return
}

func main() {
	initClients()
	http.HandleFunc("/", handleHello)
	http.HandleFunc(fmt.Sprintf("/%s/%s", TEAM_CODE, CREATE), handleCreateTeamCode)
	http.HandleFunc(fmt.Sprintf("/%s/%s", TEAM_CODE, JOIN), handleJoinTeamCode)
	http.Handle(fmt.Sprintf("/%s", TURN), websocket.Handler(handleConnection))
	http.Handle(fmt.Sprintf("/%s", USERS), websocket.Handler(handleUsersConnections))
	go handleMessage()

	fmt.Println("serving at http://localhost:8080....")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}