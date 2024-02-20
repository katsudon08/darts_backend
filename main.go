package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"golang.org/x/net/websocket"
)

var users = make(map[string]UsersData)
var teamcodes = []string{}
var teamturn = make(map[string]string)
var clients = make(map[string]map[*websocket.Conn]string)
var broadcast = make(chan Data)

func initClients() {
	clients[TURN] = make(map[*websocket.Conn]string)
	clients[USERS] = make(map[*websocket.Conn]string)
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

func deleteSliceFactor(array []string, key int) (result []string){
	tmp := []string{}
	tmp = append(tmp, array[:key]...)
	result = append(tmp, array[key+1:]...)
	return
}

func deleteTeamCodeFromClientsDiff(mux string, ws *websocket.Conn) {
	flag := true
	delete(clients[mux], ws)
	for key, teamcodesValue := range teamcodes {
		for _, clientsValue := range clients[mux] {
			if teamcodesValue == clientsValue {
				flag = false
			}
		}
		if flag {
			if mux == USERS {
				delete(users, teamcodes[key])
			}
			teamcodes = deleteSliceFactor(teamcodes, key)
			fmt.Println("teamcodes", teamcodes)
			return
		}
	}
}

func createMessageFromUsers(users []UserData) (msg string) {
	msg = ""
	for _, user := range users {
		msg += fmt.Sprintf("%s%s%s ", user.GroupNum, MARK, user.UserName)
	}
	return
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
				go deleteTeamCodeFromClientsDiff(TURN, ws)
				return
			}
			panic(err)
		}

		fmt.Println(msg)
		splittedMsg := strings.Split(msg, MARK)

		teamcode := splittedMsg[0]
		fmt.Println("teamcode:", teamcode)

		clients[TURN][ws] = teamcode
		fmt.Println("turn teamcode:",clients[TURN][ws])

		if len(splittedMsg) < 2 {
			data := Data{TURN, teamcode, teamturn[teamcode]}
			broadcast <- data
		} else {
			msg := splittedMsg[1]
			fmt.Println(teamcode, msg)
			teamturn[teamcode] = msg

			data := Data{TURN, teamcode, msg}
			broadcast <- data
		}
	}
}

// ユーザーのコネクション

func handleUsersConnection(ws *websocket.Conn) {
	defer ws.Close()

	clients[USERS][ws] = ""

	fmt.Println("users_clients", clients[USERS])

	for {
		msg := ""
		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			if err.Error() == "EOF" {
				go deleteTeamCodeFromClientsDiff(USERS, ws)
				return
			}
			panic(err)
		}

		fmt.Println(msg)
		splittedMsg := strings.Split(msg, MARK)
		fmt.Println(splittedMsg)

		teamcode := splittedMsg[0]
		fmt.Println("teamcode:", teamcode)

		clients[USERS][ws] = teamcode
		fmt.Println("user teamcode:", clients[USERS][ws])

		if len(splittedMsg) < 3 {
			sort.Sort(users[teamcode])
			fmt.Println("get users data:", users[teamcode])
			msg = createMessageFromUsers(users[teamcode])
			data := Data{USERS, teamcode, msg}
			broadcast <- data
		} else {
			num, user := splittedMsg[1], splittedMsg[2]
			fmt.Println(teamcode, num, user)

			userData := UserData{ws, num, user}

			flagForUnaddedUser := true

			for key, value := range users[teamcode] {
				if value.CheckUserWsIsCorrect(userData) {
					flagForUnaddedUser = false
					users[teamcode][key] = userData
				}
			}

			if flagForUnaddedUser {
				users[teamcode] = append(users[teamcode], userData)
			}

			sort.Sort(users[teamcode])
			fmt.Println("create or change users data:", users[teamcode])
			msg = createMessageFromUsers(users[teamcode])
			data := Data{USERS, teamcode, msg}
			broadcast <- data
		}
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
	http.Handle(fmt.Sprintf("/%s", USERS), websocket.Handler(handleUsersConnection))
	go handleMessage()

	fmt.Println("serving at http://localhost:8080....")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}