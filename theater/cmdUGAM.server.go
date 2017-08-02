package theater

import (
	"github.com/SpencerSharkey/GoFesl/GameSpy"

	"github.com/SpencerSharkey/GoFesl/lib"
	"github.com/SpencerSharkey/GoFesl/log"
)

// UGAM - SERVER Called to udpate serverquery ifo
func (tM *TheaterManager) UGAM(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	gameID := event.Command.Message["GID"]

	gdata := new(lib.RedisObject)
	gdata.New(tM.redis, "gdata", gameID)

	log.Noteln("Updating GameServer " + gameID)

	var args []interface{}

	keys := 0
	for index, value := range event.Command.Message {
		if index == "TID" {
			continue
		}

		keys++

		// Strip quotes
		if len(value) > 0 && value[0] == '"' {
			value = value[1:]
		}
		if len(value) > 0 && value[len(value)-1] == '"' {
			value = value[:len(value)-1]
		}

		gdata.Set(index, value)
		args = append(args, gameID)
		args = append(args, index)
		args = append(args, value)
	}
	_, err := tM.stmtUpdateGame.Exec(event.Command.Message["GID"], Shard)
	if err != nil {
		log.Panicln(err)
	}

	_, err = tM.setServerStatsStatement(keys).Exec(args...)
	if err != nil {
		log.Errorln("Failed to update stats for game server "+gameID, err.Error())
	}
}
