package main

type Data struct {
	Key string
	TeamCode string
	Msg string
}

type teamcodeJSON struct {
	Teamcode string `json:"teamcode"`
}

const (
	NUMBER_OF_TEAM = 6
	TURN = "turn"
	TEAM_CODE = "team-code"
	CREATE = "create"
	JOIN = "join"
	USERS = "users"
	MARK = "[:::]"
)