package fesl

import (
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// NuGetAccount - General account information retrieved, based on parameters sent
func (fM *FeslManager) NuGetAccount(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuGetAccount"
	loginPacket["heroName"] = event.Client.RedisState.Get("username")
	loginPacket["nuid"] = event.Client.RedisState.Get("email")
	loginPacket["DOBDay"] = "1"
	loginPacket["DOBMonth"] = "1"
	loginPacket["DOBYear"] = "2017"
	loginPacket["userId"] = event.Client.RedisState.Get("uID")
	loginPacket["globalOptin"] = "0"
	loginPacket["thidPartyOptin"] = "0"
	loginPacket["language"] = "enUS"
	loginPacket["country"] = "US"
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}
