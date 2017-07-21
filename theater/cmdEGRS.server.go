package theater

import (
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// EGRS - SERVER sent up, tell us if client is 'allowed' to join
func (tM *TheaterManager) EGRS(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		return
	}

	log.Noteln("wpwww")

	answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	event.Client.WriteFESL("EGRS", answer, 0x0)
}
