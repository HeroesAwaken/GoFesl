package theater

import (
	"strconv"

	"github.com/ReviveNetwork/GoFesl/lib"
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// UPLA - SERVER presumably "update player"? valid response reqiured
func (tM *TheaterManager) UPLA(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		return
	}

	var args []interface{}

	keys := 0

	pid := event.Command.Message["PID"]
	gid := event.Command.Message["GID"]

	for index, value := range event.Command.Message {
		if index == "TID" || index == "PID" || index == "GID" {
			continue
		}

		keys++

		// Strip quotes
		if len(value) > 0 && value[0] == '"' {
			value = value[1:]
		}
		if len(value) > 0 && value[len(value)-1] == '"' {
			value = value[:len(value)-1]
		}

		args = append(args, gid)
		args = append(args, pid)
		args = append(args, index)
		args = append(args, value)
	}

	var err error
	_, err = tM.setServerPlayerStatsStatement(keys).Exec(args...)
	if err != nil {
		log.Errorln("Failed to update stats for player "+pid, err.Error())
	}

	gdata := new(lib.RedisObject)
	gdata.New(tM.redis, "gdata", event.Command.Message["GID"])

	num, _ := strconv.Atoi(gdata.Get("AP"))

	num++

	gdata.Set("AP", strconv.Itoa(num))

	// Don't answer
	/*answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	answer["PID"] = event.Command.Message["PID"]
	answer["P-cid"] = event.Command.Message["P-cid"]
	log.Noteln(answer)
	event.Client.WriteFESL("UPLA", answer, 0x0)
	tM.logAnswer(event.Command.Query, answer, 0x0)*/
}
