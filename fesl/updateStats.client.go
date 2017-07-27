package fesl

import (
	"strconv"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// UpdateStats - updates stats about a soldier
func (fM *FeslManager) UpdateStats(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answer := event.Command.Message
	answer["TXN"] = "UpdateStats"

	userId := event.Client.RedisState.Get("uID")

	users, _ := strconv.Atoi(event.Command.Message["u.[]"])
	for i := 0; i < users; i++ {
		owner, ok := event.Command.Message["u."+strconv.Itoa(i)+".o"]

		if !ok {
			return
		}

		// Generate our argument list for the statement -> userId, owner, key1, value1, userId, owner, key2, value2, userId, owner, ...
		var args []interface{}
		keys, _ := strconv.Atoi(event.Command.Message["u."+strconv.Itoa(i)+".s.[]"])
		for j := 0; j < keys; j++ {

			key := event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".k"]
			value := event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".t"]

			if value == "" {
				value = event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".v"]
			}

			// We need to append 3 values for each insert/update,
			// owner, key and value
			log.Noteln("Updating stats:", userId, owner, key, value)
			args = append(args, userId)
			args = append(args, owner)
			args = append(args, key)
			args = append(args, value)
		}

		_, err := fM.setStatsStatement(keys).Exec(args...)
		if err != nil {
			log.Errorln("Failed setting stats for hero "+owner, err.Error())
		}
	}

	event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)
}
