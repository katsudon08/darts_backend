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
var teamcodeToGamesData = make(map[string]GamesData)
var teamcodeToOrderNum = make(map[string]int)
var broadcast = make(chan Data)

func initClients() {
	clients[TURN] = make(map[*websocket.Conn]string)
	clients[USERS] = make(map[*websocket.Conn]string)
	clients[TRANSITION] = make(map[*websocket.Conn]string)
	clients[GAME] = make(map[*websocket.Conn]string)
}

func getTeamNum(teamcode string) (teamNum int) {
	teamNum = 0
	for _, value := range clients[USERS] {
		if value == teamcode {
			teamNum++
		}
	}
	return
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

		teamNum := getTeamNum(teamcode)

		for _, value := range teamcodes {
			if value == teamcode && teamNum < 6 {
				_, err := w.Write([]byte(teamcode))
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

func deleteStringSliceFactor(array []string, key int) (result []string) {
	tmp := []string{}
	tmp = append(tmp, array[:key]...)
	result = append(tmp, array[key+1:]...)
	return
}

func deleteUserDataSliceFactor(array []UserData, key int) (result []UserData) {
	tmp := []UserData{}
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
			teamcodes = deleteStringSliceFactor(teamcodes, key)
			fmt.Println("teamcodes", teamcodes)
			return
		}
	}
}

func deleteUserFromUsers(teamcode string, ws *websocket.Conn) {
	for key, value := range users[teamcode] {
		if value.Ws == ws {
			users[teamcode] = deleteUserDataSliceFactor(users[teamcode], key)
			return
		}
	}
}

func createMessageFromUsers(users UsersData) (msg string) {
	msg = ""
	for i, user := range users {
		if i < users.Len()-1 {
			msg += fmt.Sprintf("%s%s%s ", user.GroupNum, MARK, user.UserName)
		} else {
			msg += fmt.Sprintf("%s%s%s", user.GroupNum, MARK, user.UserName)
		}
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

		fmt.Println("msg:", msg)
		splittedMsg := strings.Split(msg, MARK)
		fmt.Println(splittedMsg)

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

	fmt.Println("users_clients:", clients[USERS])

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

		fmt.Println("msg:", msg)
		splittedMsg := strings.Split(msg, MARK)
		fmt.Println(splittedMsg)

		teamcode := splittedMsg[0]
		fmt.Println("teamcode:", teamcode)

		clients[USERS][ws] = teamcode
		fmt.Println("user teamcode:", clients[USERS][ws])

		teamNum := getTeamNum(teamcode)

		if teamNum > 6 {
			data := Data{USERS, teamcode, " "}
			broadcast <- data
			continue
		}

		if len(splittedMsg) < 2 {
			sort.Sort(users[teamcode])
			fmt.Println("get users data:", users[teamcode])
			msg = createMessageFromUsers(users[teamcode])
			data := Data{USERS, teamcode, msg}
			broadcast <- data
		} else if len(splittedMsg) < 3 {
			deleteUserFromUsers(teamcode, ws)
			sort.Sort(users[teamcode])
			fmt.Println("deleted users data:", users[teamcode])
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

// ゲーム画面への遷移を管理するコネクション

func handleTransitionToGameScreen(ws *websocket.Conn) {
	defer ws.Close()

	clients[TRANSITION][ws] = ""

	fmt.Println("transition_clients:", clients[GAME])

	for {
		teamcode := ""
		err := websocket.Message.Receive(ws, &teamcode)
		if err != nil {
			if err.Error() == "EOF" {
				go deleteTeamCodeFromClientsDiff(TRANSITION, ws)
				return
			}
		}

		fmt.Println("teamcode:", teamcode)
		clients[TRANSITION][ws] = teamcode

		data := Data{TRANSITION, teamcode, ""}
		broadcast <- data
	}
}

// ゲームのコネクション

func handleGameConnection(ws * websocket.Conn) {
	defer ws.Close()

	clients[GAME][ws] = ""

	fmt.Println("game_clients:", clients[GAME])

	for {
		msg := ""
		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			if err.Error() == "EOF" {
				go deleteTeamCodeFromClientsDiff(GAME, ws)
				return
			}
			panic(err)
		}

		fmt.Println("msg:", msg)
		splittedMsg := strings.Split(msg, MARK)
		fmt.Println(splittedMsg)

		teamcode, groupNum, userName, userId := splittedMsg[0], splittedMsg[1], splittedMsg[2], splittedMsg[3]
		fmt.Println("teamcode:", teamcode)

		clients[GAME][ws] = teamcode
		fmt.Println("game teamcode:", clients[GAME][ws])

		gameData := GameData{teamcode, groupNum, userName, userId}

		if len(splittedMsg) < 5 {
			teamcodeToGamesData[teamcode] =  append(teamcodeToGamesData[teamcode], gameData)
			fmt.Printf("games_data[%s]:%v",teamcode, teamcodeToGamesData[teamcode])

			teamcodeToOrderNum[teamcode] = 0
			sort.Sort(teamcodeToGamesData[teamcode])

			// TODO: 一番最初のプレーヤーに操作権を与える
		} else {
			score := splittedMsg[4]
			fmt.Println(score)

			// TODO: 操作権を次のプレイヤーに移し、どのチームに何点加算するのかを送信
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
	http.Handle(fmt.Sprintf("/%s", TRANSITION), websocket.Handler(handleTransitionToGameScreen))
	http.Handle(fmt.Sprintf("/%s", GAME), websocket.Handler(handleGameConnection))
	go handleMessage()

	fmt.Println("serving at http://localhost:8080....")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}