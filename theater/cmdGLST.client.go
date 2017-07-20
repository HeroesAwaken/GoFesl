package theater

import (
	"github.com/ReviveNetwork/GoFesl/GameSpy"
	"github.com/ReviveNetwork/GoFesl/log"
)

// GLST - CLIENT called to get a list of game servers? Irrelevant for heroes.
func (tM *TheaterManager) GLST(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	log.Noteln("GLST was called")
}
