package theater

import (
	"github.com/ReviveNetwork/GoFesl/GameSpy"
	"github.com/ReviveNetwork/GoFesl/log"
)

// USER - SHARED Called to get user data about client? No idea
func (tM *TheaterManager) USER(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	answer["NAME"] = "GenericUser"
	answer["CID"] = ""
	event.Client.WriteFESL(event.Command.Query, answer, 0x0)
	tM.logAnswer(event.Command.Query, answer, 0x0)
}
