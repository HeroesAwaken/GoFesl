package theater

import (
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// UBRA - SERVER Called to  update server data
func (tM *TheaterManager) UBRA(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	// Just acknoledge for now, we need to udpate redis though.
	answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	event.Client.WriteFESL(event.Command.Query, answer, 0x0)
	tM.logAnswer(event.Command.Query, answer, 0x0)
}
