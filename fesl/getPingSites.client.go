package fesl

import (
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// GetPingSites - returns a list of endpoints to test for the lowest latency on a client
func (fM *FeslManager) GetPingSites(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	// gva = eu central
	// nrt = us east

	answer := make(map[string]string)
	answer["TXN"] = "GetPingSites"
	answer["minPingSitesToPing"] = "2"
	answer["pingSites.[]"] = "2"
	answer["pingSites.0.addr"] = "45.77.66.233"
	answer["pingSites.0.name"] = "gva"
	answer["pingSites.0.type"] = "0"
	answer["pingSites.1.addr"] = "45.77.76.193"
	answer["pingSites.1.name"] = "nrt"
	answer["pingSites.1.type"] = "0"

	event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)
}
