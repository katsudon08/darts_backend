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
	clients[GAME_DISPLAY] = make(map[*websocket.Conn]string)
	clients[RESULT] = make(map[*websocket.Conn]string)
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

		fmt.Println("access request")

		buf := new(bytes.Buffer)
		io.Copy(buf, body)

		var data teamcodeJSON
		json.Unmarshal(buf.Bytes(), &data)

		fmt.Println("data:", data)

		teamcode := data.Teamcode
		fmt.Println("teamcode_join:", teamcode)
		fmt.Println(r.Host)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", fmt.Sprintf("http://%s, https://%s", r.Host, r.Host))
		w.Header().Set("Access-Control-Allow-Methods", "POST")

		teamNum := getTeamNum(teamcode)

		flag := true

		for _, value := range teamcodes {
			if value == teamcode && teamNum < 6 {
				_, err := w.Write([]byte(teamcode))
				fmt.Println("success")
				if err != nil {
					panic(err)
				}
				flag = false
			}
		}

		if flag {
			_, err := w.Write([]byte(""))
			if err != nil {
				panic(err)
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

func ReceiveWebsocketMessage(ws *websocket.Conn, key string) (msg string) {
	msg = ""
	err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			if err.Error() == "EOF" {
				go deleteTeamCodeFromClientsDiff(key, ws)
				return CANCEL
			}
			panic(err)
		}

		fmt.Println("msg:", msg)
		return
}

// ターンのコネクション

func handleTurnConnection(ws *websocket.Conn) {
	defer ws.Close()

	clients[TURN][ws] = ""

	fmt.Println("turn_clients:", clients[TURN])

	for {
		msg := ReceiveWebsocketMessage(ws, TURN)
		if msg == CANCEL {
			return
		}
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
		msg := ReceiveWebsocketMessage(ws, USERS)
		if msg == CANCEL {
			return
		}
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
			fmt.Printf("\n")
			msg = createMessageFromUsers(users[teamcode])
			fmt.Println("msg:", msg)
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

func handleTransitionToGameScreenConnection(ws *websocket.Conn) {
	defer ws.Close()

	clients[TRANSITION][ws] = ""

	fmt.Println("transition_clients:", clients[TRANSITION])

	for {
		fmt.Println("-------------transition-----------------")
		msg := ReceiveWebsocketMessage(ws, TRANSITION)
		if msg == CANCEL {
			return
		}
		splittedMsg := strings.Split(msg, " ")
		teamcode := splittedMsg[0]

		fmt.Println("transition_teamcode:", teamcode)
		clients[TRANSITION][ws] = teamcode

		fmt.Println(clients[TRANSITION])

		fmt.Println("-------------transition-----------------")

		if len(splittedMsg) < 2 {
			data := Data{TRANSITION, teamcode, ""}
			broadcast <- data
		}
	}
}

func createMessageFromGameData(gameData GameData, nextUserId string, score string, isLast bool) (msg string) {
	msg = gameData.GroupNum + MARK + gameData.UserId + MARK + nextUserId + MARK + score + MARK
	if isLast {
		msg += "isLast"
	}
	return
}

// ゲームのコネクション

func handleGameConnection(ws * websocket.Conn) {
	defer ws.Close()

	clients[GAME][ws] = ""

	fmt.Println("game_clients:", clients[GAME])

	for {
		msg := ReceiveWebsocketMessage(ws, GAME)
		if msg == CANCEL {
			return
		}
		splittedMsg := strings.Split(msg, MARK)
		fmt.Println("message:", splittedMsg)

		teamcode, groupNum, userName, userId := splittedMsg[0], splittedMsg[1], splittedMsg[2], splittedMsg[3]
		fmt.Println("teamcode:", teamcode)

		clients[GAME][ws] = teamcode
		fmt.Println("game teamcode:", clients[GAME][ws])

		fmt.Println("teamcodeToGameData:", teamcodeToGamesData[teamcode])

		gameData := GameData{teamcode, groupNum, userName, userId}

		if len(splittedMsg) < 5 {
			teamcodeToGamesData[teamcode] =  append(teamcodeToGamesData[teamcode], gameData)
			fmt.Printf("games_data[%s]:%v\n",teamcode, teamcodeToGamesData[teamcode])

			teamcodeToOrderNum[teamcode] = 0
			sort.Sort(teamcodeToGamesData[teamcode])

			// TODO: 一番最初のプレーヤーに操作権を与える
			fmt.Println(teamcodeToGamesData[teamcode][0].UserId)

			msg := teamcodeToGamesData[teamcode][0].UserId
			data := Data{GAME, gameData.Teamcode, msg}

			broadcast <- data
		} else {
			score := splittedMsg[4]
			fmt.Println("score:", score)

			// TODO: 操作権を次のプレイヤーに移し、どのチームに何点加算するのかを送信

			lastIndex := len(teamcodeToGamesData[teamcode]) - 1
			fmt.Println("lastIndex:", lastIndex)
			fmt.Println(teamcodeToGamesData[teamcode][lastIndex].UserId)
			isLast := gameData.UserId == teamcodeToGamesData[teamcode][lastIndex].UserId
			fmt.Println(isLast)

			fmt.Println("games data:", teamcodeToGamesData[teamcode])

			nextIndex := teamcodeToOrderNum[teamcode] + 1
			fmt.Println("lastIndex:", lastIndex)
			fmt.Println("nextIndex:", nextIndex)

			var nextGameData GameData
			if (isLast) {
				nextGameData = teamcodeToGamesData[teamcode][0]
				fmt.Printf("debag 1\n\n")
				fmt.Println("next game data:", nextGameData)
				teamcodeToOrderNum[teamcode] = 0
			} else {
				nextGameData = teamcodeToGamesData[teamcode][nextIndex]
				fmt.Printf("debag 2\n\n")
				fmt.Println("next game data:", nextGameData)
				teamcodeToOrderNum[teamcode]++
			}


			// groupNum userId nextUserId score isLast
			msg := createMessageFromGameData(gameData, nextGameData.UserId, score, isLast)
			fmt.Println("msg:", msg)
			data := Data{GAME, gameData.Teamcode, msg}
			broadcast <- data
		}
	}
}

func createMessageFromDisplayMessage(userGroup string, userName string, score string) (msg string) {
	msg = userGroup + MARK + userName + MARK + score
	return
}

// ゲーム画面の表示を管理するコネクション

func handleGameDisplayConnection(ws *websocket.Conn) {
	defer ws.Close()

	clients[GAME_DISPLAY][ws] = ""

	fmt.Println("turn_clients:", clients[GAME_DISPLAY])

	for {
		msg := ReceiveWebsocketMessage(ws, GAME_DISPLAY)
		if msg == CANCEL {
			return
		}
		fmt.Println(msg)
		splittedMsg := strings.Split(msg, MARK)

		teamcode, groupNum, userName, score := splittedMsg[0], splittedMsg[1], splittedMsg[2], splittedMsg[3]
		fmt.Println("teamcode:", teamcode)
		clients[GAME_DISPLAY][ws] = teamcode

		if len(splittedMsg) < 5 {
			msg = createMessageFromDisplayMessage(groupNum, userName, score)
			data := Data{GAME_DISPLAY, teamcode, msg}
			broadcast <- data
		} else {
			firstGameData := teamcodeToGamesData[teamcode][0]
			msg = createMessageFromDisplayMessage(firstGameData.GroupNum, firstGameData.UserName, score)
			data := Data{GAME_DISPLAY, teamcode, msg}
			broadcast <- data
		}
	}
}

// リザルト画面を管理するコネクション

func handleResultConnection(ws *websocket.Conn) {
	defer ws.Close()

	clients[RESULT][ws] = ""

	fmt.Println("turn_clients:", clients[RESULT])

	for {
		msg := ReceiveWebsocketMessage(ws, RESULT)
		if msg == CANCEL {
			return
		}
		fmt.Println(msg)
		splittedMsg := strings.Split(msg, MARK)

		teamcode, groupNum, score := splittedMsg[0], splittedMsg[1], splittedMsg[2]
		fmt.Println("teamcode:", teamcode)
		teamcodeToGamesData[teamcode] = nil

		clients[RESULT][ws] = teamcode

		msg = groupNum + MARK + score

		data := Data{RESULT, teamcode, msg}
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
	http.Handle(fmt.Sprintf("/%s", USERS), websocket.Handler(handleUsersConnection))
	http.Handle(fmt.Sprintf("/%s", TRANSITION), websocket.Handler(handleTransitionToGameScreenConnection))
	http.Handle(fmt.Sprintf("/%s", GAME), websocket.Handler(handleGameConnection))
	http.Handle(fmt.Sprintf("/%s", GAME_DISPLAY), websocket.Handler(handleGameDisplayConnection))
	http.Handle(fmt.Sprintf("/%s", RESULT), websocket.Handler(handleResultConnection))
	go handleMessage()

	fmt.Println("serving at http://localhost:8080....")
	log.Fatal(http.ListenAndServe(":8080", nil))
}