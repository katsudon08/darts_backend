package main

import "golang.org/x/net/websocket"

type Data struct {
	Key string
	TeamCode string
	Msg string
}

type UserData struct {
	Ws *websocket.Conn
	GroupNum string
	UserName string
}

type UsersData []UserData

func (u *UserData) CheckUserWsIsCorrect(user UserData) bool {
	return u.Ws == user.Ws
}

// 以下はソートを行うために満たす必要があるメソッド群

func (u UsersData) Len() int {
	return len(u)
}

func (u UsersData) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

func (u UsersData) Less(i, j int) bool {
	return u[i].GroupNum < u[j].GroupNum
}

type GameData struct {
	Teamcode string
	GroupNum string
	UserName string
	UserId string
}

type GamesData []GameData

// 以下はソートを行うために満たす必要があるメソッド群

func (g GamesData) Len() int {
	return len(g)
}

func (g GamesData) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

func (g GamesData) Less(i, j int) bool {
	return g[i].GroupNum < g[j].GroupNum
}

type teamcodeJSON struct {
	Teamcode string `json:"teamcode"`
}

const (
	TURN = "turn"
	USERS = "users"
	TRANSITION = "transition"
	GAME = "game"
	GAME_DISPLAY = "game-display"
	NUMBER_OF_TEAM = 6
	TEAM_CODE = "team-code"
	CREATE = "create"
	JOIN = "join"
	MARK = "[:::]"
)