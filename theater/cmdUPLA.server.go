package theater

import (
	"github.com/ReviveNetwork/GoFesl/GameSpy"
	"github.com/ReviveNetwork/GoFesl/log"
)

// UPLA - SERVER presumably "update player"? valid response reqiured
func (tM *TheaterManager) UPLA(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		return
	}

	answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	answer["PID"] = event.Command.Message["PID"]
	answer["P-cid"] = event.Command.Message["P-cid"]
	log.Noteln(answer)
	event.Client.WriteFESL("UPLA", answer, 0x0)
}
