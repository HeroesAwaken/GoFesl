package theater

import (
	"github.com/HeroesAwaken/GoFesl/GameSpy"
	"github.com/HeroesAwaken/GoFesl/log"
)

// ECNL - CLIENT calls when they want to leave
func (tM *TheaterManager) ECNL(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	//wantsToLeaveQueue = true

	answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	answer["GID"] = event.Command.Message["GID"]
	answer["LID"] = event.Command.Message["LID"]
	event.Client.WriteFESL("ECNL", answer, 0x0)
	tM.logAnswer("ECNL", answer, 0x0)

	/*ap := make(map[string]string)
	ap["TID"] = "7"
	ap["GID"] = "5459"
	ap["LID"] = "1"
	event.Client.WriteFESL("ECNLmisc", ap, 0x0)
	tM.logAnswer("ECNLmisc", ap, 0x0)		*/
}
