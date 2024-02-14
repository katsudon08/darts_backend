package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"golang.org/x/net/websocket"
)

type teamcodeJSON struct {
	Teamcode string `json:"teamcode"`
}

var users = make(map[*websocket.Conn]string)
var usersMsg = ""
var teamcodes []string
var stopChan = make(chan bool)
var clients = make(map[string]map[*websocket.Conn]bool)
var broadcast = make(chan Data)

func initClients() {
	clients[TEAM_CODE] = make(map[*websocket.Conn]bool)
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

		for _, value := range teamcodes {
			if value == teamcode {
				return
			}
		}

		stopChan <- true

		teamcodes = append(teamcodes, teamcode)
		fmt.Println(teamcodes)

		go worker(stopChan)

		_, err := w.Write([]byte(teamcode))
		if err != nil {
			log.Print(err)
		}
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

		for _, value := range teamcodes {
			if teamcode == value {
				_, err := w.Write([]byte(teamcode))
				if err != nil {
					log.Print(err)
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
			log.Print(err)
		}

		fmt.Println(msg)

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
			log.Print(err)
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
				log.Print(err)
			}
		}
	}
}

func worker(stopChan chan bool) {
	for _, value := range teamcodes {
		fmt.Println("handler", value)
		http.Handle(fmt.Sprintf("/%s/%s", TURN, value), websocket.Handler(handleConnection))
	}

	for {
		<-stopChan
		fmt.Println("stopping...")
		return
	}
}

func main() {
	initClients()
	http.HandleFunc("/", handleHello)
	http.HandleFunc(fmt.Sprintf("/%s/%s", TEAM_CODE, CREATE), handleCreateTeamCode)
	http.HandleFunc(fmt.Sprintf("/%s/%s", TEAM_CODE, JOIN), handleJoinTeamCode)
	go worker(stopChan)
	http.Handle(fmt.Sprintf("/%s", USERS), websocket.Handler(handleUsersConnections))
	go handleMessage()

	fmt.Println("serving at http://localhost:8080....")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}