package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/websocket"
)

var users = make(map[*websocket.Conn]string)
var usersMsg = ""
var teamcodes = []string{}
var clients = make(map[string]map[*websocket.Conn]string)
var broadcast = make(chan Data)

func initClients() {
	clients[TURN] = make(map[*websocket.Conn]string)
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

		for _, value := range teamcodes {
			if value == teamcode {
				return
			}
		}

		teamcodes = append(teamcodes, teamcode)

		_, err := w.Write([]byte(teamcode))
		if err != nil {
			panic(err)
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
			if value == teamcode {
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
	defer ws.Close()

	clients[TURN][ws] = ""

	fmt.Println("turn_clients:", clients[TURN])

	for {
		msg := ""

		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			// 通信切断時
			if err.Error() == "EOF" {
				flag := true
				delete(clients[TURN], ws)
				for key, teamcodesValue := range teamcodes {
					for _, clientsValue := range clients[TURN] {
						if teamcodesValue == clientsValue {
							flag = false
						}
					}
					if flag {
						tmp := []string{}
						tmp = append(tmp, teamcodes[:key-1]...)
						teamcodes = append(tmp, teamcodes[key+1:]...)
						return
					}
				}

				return
			}
			panic(err)
		}

		fmt.Println(msg)
		splittedMsg := strings.Split(msg, MARK)
		teamcode, msg := splittedMsg[0], splittedMsg[1]
		fmt.Println(teamcode, msg)

		clients[TURN][ws] = teamcode

		data := Data{TURN, teamcode, msg}

		broadcast <- data
	}
}

// ユーザーのコネクション

func handleUsersConnection(ws *websocket.Conn) {

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

// メッセージの送信

func handleMessage() {
	for {
		// broadcastからメッセージを受取る
		data := <- broadcast

		fmt.Println(fmt.Sprintf("msg/clients[%s]:", data.Key), clients[data.Key])

		// 各クライアントへのメッセージの送信
		for client, teamcode := range clients[data.Key] {
			if teamcode == data.TeamCode {
				err := websocket.Message.Send(client, data.Msg)
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

func main() {
	initClients()
	http.HandleFunc("/", handleHello)
	http.HandleFunc(fmt.Sprintf("/%s/%s", TEAM_CODE, CREATE), handleCreateTeamCode)
	http.HandleFunc(fmt.Sprintf("/%s/%s", TEAM_CODE, JOIN), handleJoinTeamCode)
	http.Handle(fmt.Sprintf("/%s", TURN), websocket.Handler(handleTurnConnection))
	http.Handle(fmt.Sprintf("/%s", USERS), websocket.Handler(handleUsersConnections))
	go handleMessage()

	fmt.Println("serving at http://localhost:8080....")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}