package fesl

import (
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// NuLoginPersona - soldier login command
func (fM *FeslManager) NuLoginPersona(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuLoginPersona"
	loginPacket["lkey"] = event.Client.RedisState.Get("keyHash")
	loginPacket["profileId"] = event.Client.RedisState.Get("uID")
	loginPacket["userId"] = event.Client.RedisState.Get("uID")
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}
