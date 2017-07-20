package theater

import (
	"github.com/ReviveNetwork/GoFesl/GameSpy"

	"github.com/ReviveNetwork/GoFesl/lib"
	"github.com/ReviveNetwork/GoFesl/log"
)

// UGAM - SERVER Called to udpate serverquery ifo
func (tM *TheaterManager) UGAM(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	gdata := new(lib.RedisObject)
	gdata.New(tM.redis, "gdata", event.Command.Message["GID"])

	log.Noteln("Updating GameServer " + event.Command.Message["GID"])

	for index, value := range event.Command.Message {
		if index == "TID" {
			continue
		}

		// Strip quotes
		if len(value) > 0 && value[0] == '"' {
			value = value[1:]
		}
		if len(value) > 0 && value[len(value)-1] == '"' {
			value = value[:len(value)-1]
		}

		gdata.Set(index, value)
	}
}
