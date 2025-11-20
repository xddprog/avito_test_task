package entity


type User struct {
	ID string `json:"id"`
	Username string `json:"username"`
	IsActive bool `json:"is_active"`
	TeamName string `json:"team_name"`
}


type SetUserIsActiveRequest struct {
	UserId string `json:"user_id"`
	IsActive bool `json:"is_active"`
}