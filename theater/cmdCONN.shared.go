package theater

import (
	"strconv"
	"time"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// CONN - SHARED (???) called on connection
func (tM *TheaterManager) CONN(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	answer["TIME"] = strconv.FormatInt(time.Now().UTC().Unix(), 10)
	answer["activityTimeoutSecs"] = "30"
	answer["PROT"] = event.Command.Message["PROT"]
	event.Client.WriteFESL(event.Command.Query, answer, 0x0)
	tM.logAnswer(event.Command.Query, answer, 0x0)
}
