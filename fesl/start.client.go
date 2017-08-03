package fesl

import (
	"strings"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// Start - a method of pnow
func (fM *FeslManager) Start(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	// Check if user is allowed to matchmake
	if !fM.userHasPermission(event.Client.RedisState.Get("uID"), "game.matchmake") {
		log.Noteln("User not worthy: " + event.Client.RedisState.Get("username"))
		return
	}

	// Check if user has op rocket equipped
	rows, err := fM.getStatsStatement(2).Query(event.Client.RedisState.Get("heroID"), event.Client.RedisState.Get("uID"), "c_eqp", "c_apr")
	if err != nil {
		log.Errorln("Failed gettings stats for hero "+event.Client.RedisState.Get("heroID"), err.Error())
	}

	stats := make(map[string]string)
	for rows.Next() {
		var userID, heroID, statsKey, statsValue string
		err := rows.Scan(&userID, &heroID, &statsKey, &statsValue)
		if err != nil {
			log.Errorln("Issue with database:", err.Error())
		}
		stats[statsKey] = statsValue
	}

	if strings.Contains(stats["c_eqp"], "3018") {
		log.Noteln("User trying to matchmake with op launcher")
		return
	}

	log.Noteln("START CALLED")
	log.Noteln(event.Command.Message["partition.partition"])
	answer := make(map[string]string)
	answer["TXN"] = "Start"
	answer["id.id"] = "1"
	answer["id.partition"] = event.Command.Message["partition.partition"]
	event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)

	fM.Status(event)
}
