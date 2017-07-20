package theater

import (
	"github.com/ReviveNetwork/GoFesl/GameSpy"
	"github.com/ReviveNetwork/GoFesl/lib"
	"github.com/ReviveNetwork/GoFesl/log"
)

// GDAT - CLIENT called to get data about the server
func (tM *TheaterManager) GDAT(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	gameID := event.Command.Message["GID"]

	gameServer := new(lib.RedisObject)
	gameServer.New(tM.redis, "gdata", gameID)

	answer := make(map[string]string)

	answer["TID"] = event.Command.Message["TID"]

	for _, dataKey := range gameServer.HKeys() {
		// Strip quotes
		if len(dataKey) > 0 && dataKey[0] == '"' {
			dataKey = dataKey[1:]
		}
		if len(dataKey) > 0 && dataKey[len(dataKey)-1] == '"' {
			dataKey = dataKey[:len(dataKey)-1]
		}

		answer[dataKey] = gameServer.Get(dataKey)
	}

	event.Client.WriteFESL("GDAT", answer, 0x0)
	tM.logAnswer("GDAT", answer, 0x0)

}
