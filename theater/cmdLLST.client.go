package theater

import (
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// LLST - CLIENT (???) unknown, potentially bookmarks
func (tM *TheaterManager) LLST(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	answer["NUM-LOBBIES"] = "1"
	event.Client.WriteFESL(event.Command.Query, answer, 0x0)

	// Todo: create dataset for lobbies, iterate through and send one for each lobby (LDAT>)()
	ldatPacket := make(map[string]string)
	ldatPacket["TID"] = "5"
	ldatPacket["FAVORITE-GAMES"] = "0"
	ldatPacket["FAVORITE-PLAYERS"] = "0"
	ldatPacket["LID"] = "1"
	ldatPacket["LOCALE"] = "en_US"
	ldatPacket["MAX-GAMES"] = "10000"
	ldatPacket["NAME"] = "bfwestPC02"
	ldatPacket["NUM-GAMES"] = "1"
	ldatPacket["PASSING"] = "0"
	event.Client.WriteFESL("LDAT", ldatPacket, 0x0)
	tM.logAnswer("LDAT", ldatPacket, 0x0)
}
