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

	users, _ := strconv.Atoi(event.Command.Message["u.[]"])
	for i := 0; i < users; i++ {
		query := ""
		owner, ok := event.Command.Message["u."+strconv.Itoa(i)+".o"]

		if !ok {
			return
		}

		statsNum, _ := strconv.Atoi(event.Command.Message["u."+strconv.Itoa(i)+".s.[]"])
		for j := 0; j < statsNum; j++ {
			if event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".t"] != "" {
				query += event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".k"] + "='" + MysqlRealEscapeString(event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".t"]) + "', "
			} else {
				// TODO: Needs to be fixed, v = change, so v = -1 means substract one
				query += event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".k"] + "='" + MysqlRealEscapeString(event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".v"]) + "', "
			}
		}

		if owner != "0" && owner != event.Client.RedisState.Get("uID") {
			sql := "UPDATE `awaken_heroes_stats` SET " + query + "pid=" + owner + " WHERE pid = " + owner + ""
			_, err := fM.db.Exec(sql)
			if err != nil {
				log.Errorln(err)
			}
		} else {
			sql := "UPDATE `awaken_heroes_accounts` SET " + query + "uid=" + owner + " WHERE uid = " + owner + ""
			_, err := fM.db.Exec(sql)
			if err != nil {
				log.Errorln(err)
			}
		}
	}

	event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)
}
