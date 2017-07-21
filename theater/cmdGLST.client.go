package theater

import (
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// GLST - CLIENT called to get a list of game servers? Irrelevant for heroes.
func (tM *TheaterManager) GLST(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	log.Noteln("GLST was called")
}
