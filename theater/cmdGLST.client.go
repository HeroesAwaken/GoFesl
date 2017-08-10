package theater

import (
	"github.com/HeroesAwaken/GoFesl/GameSpy"
	"github.com/HeroesAwaken/GoFesl/log"
)

// GLST - CLIENT called to get a list of game servers? Irrelevant for heroes.
func (tM *TheaterManager) GLST(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	log.Noteln("GLST was called")
}
