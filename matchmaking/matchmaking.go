package matchmaking

import "github.com/ReviveNetwork/GoFesl/GameSpy"

// Games - a list of available games
var Games = make(map[string]*GameSpy.Client)

// FindAvailableGID - returns a GID suitable for the player to join (ADD A PID HERE)
func FindAvailableGID() string {

	var gameID string

	for k := range Games {
		gameID = k
	}

	return gameID
}
