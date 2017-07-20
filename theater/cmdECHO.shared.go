package theater

import (
	"strconv"

	"github.com/ReviveNetwork/GoFesl/GameSpy"
	"github.com/ReviveNetwork/GoFesl/log"
)

// ECHO - SHARED called like some heartbeat
func (tM *TheaterManager) ECHO(event GameSpy.SocketUDPEvent) {
	command := event.Data.(*GameSpy.CommandFESL)

	answer := make(map[string]string)
	answer["TID"] = command.Message["TID"]
	answer["TXN"] = command.Message["TXN"]
	answer["IP"] = event.Addr.IP.String()
	answer["PORT"] = strconv.Itoa(event.Addr.Port)
	answer["ERR"] = "0"
	answer["TYPE"] = "1"
	err := tM.socketUDP.WriteFESL("ECHO", answer, 0x0, event.Addr)
	if err != nil {
		log.Errorln(err)
	}
	tM.logAnswer("ECHO", answer, 0x0)
}
