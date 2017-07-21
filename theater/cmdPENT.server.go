package theater

import "github.com/SpencerSharkey/GoFesl/GameSpy"

// PENT - SERVER sent up when a player joins (entitle player?)
func (tM *TheaterManager) PENT(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		return
	}

	// This allows all right now, I think.
	answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	answer["PID"] = event.Command.Message["PID"]
	event.Client.WriteFESL("PENT", answer, 0x0)
}
