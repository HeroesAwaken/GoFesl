package theater

import (
	"github.com/HeroesAwaken/GoFesl/GameSpy"
	"github.com/HeroesAwaken/GoFesl/log"
)

// PENT - SERVER sent up when a player joins (entitle player?)
func (tM *TheaterManager) PLVT(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		return
	}

	pid := event.Command.Message["PID"]

	// Get 4 stats for PID
	rows, err := tM.getStatsStatement(4).Query(pid, "c_kit", "c_team", "elo", "level")
	if err != nil {
		log.Errorln("Failed gettings stats for hero "+pid, err.Error())
	}

	stats := make(map[string]string)

	for rows.Next() {
		var userID, heroID, heroName, statsKey, statsValue string
		err := rows.Scan(&userID, &heroID, &heroName, &statsKey, &statsValue)
		if err != nil {
			log.Errorln("Issue with database:", err.Error())
		}
		stats[statsKey] = statsValue
	}

	switch stats["c_team"] {
	case "1":
		_, err = tM.stmtGameDecreaseTeam1.Exec(event.Command.Message["GID"], Shard)
		if err != nil {
			log.Panicln(err)
		}
	case "2":
		_, err = tM.stmtGameDecreaseTeam2.Exec(event.Command.Message["GID"], Shard)
		if err != nil {
			log.Panicln(err)
		}
	default:
		log.Errorln("Invalid team " + stats["c_team"] + " for " + pid)
	}

	answer := make(map[string]string)
	answer["PID"] = event.Command.Message["PID"]
	answer["LID"] = event.Command.Message["LID"]
	answer["GID"] = event.Command.Message["GID"]
	event.Client.WriteFESL("KICK", answer, 0x0)

	answer = make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	event.Client.WriteFESL("PLVT", answer, 0x0)
}
