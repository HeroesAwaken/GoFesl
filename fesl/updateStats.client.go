package fesl

import (
	"strconv"
	"strings"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

type stat struct {
	text  string
	value float64
}

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
		if event.Client.RedisState.Get("clientType") == "server" {

			var id, userIDhero, heroName, online string
			err := fM.stmtGetHeroeByID.QueryRow(owner).Scan(&id, &userIDhero, &heroName, &online)
			if err != nil {
				log.Noteln("Persona not worthy!")
				return
			}

			userId = userIDhero
			log.Noteln("Server updating stats")
		}

		if !ok {
			return
		}

		stats := make(map[string]*stat)

		// Get current stats from DB
		// Generate our argument list for the statement -> heroID, userID, key1, key2, key3, ...
		var argsGet []interface{}
		statsKeys := make(map[string]string)
		argsGet = append(argsGet, owner)
		argsGet = append(argsGet, userId)
		keys, _ := strconv.Atoi(event.Command.Message["u."+strconv.Itoa(i)+".s.[]"])
		for j := 0; j < keys; j++ {
			argsGet = append(argsGet, event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".k"])
			statsKeys[event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".k"]] = strconv.Itoa(j)
		}

		rows, err := fM.getStatsStatement(keys).Query(argsGet...)
		if err != nil {
			log.Errorln("Failed gettings stats for hero "+owner, err.Error())
		}

		count := 0
		for rows.Next() {
			var userID, heroID, statsKey, statsValue string
			err := rows.Scan(&userID, &heroID, &statsKey, &statsValue)
			if err != nil {
				log.Errorln("Issue with database:", err.Error())
			}

			intValue, err := strconv.ParseFloat(statsValue, 64)
			if err != nil {
				intValue = 0
			}
			stats[statsKey] = &stat{
				text:  statsValue,
				value: intValue,
			}

			delete(statsKeys, statsKey)
			count++
		}

		// Send stats not found with default value of ""
		for key := range statsKeys {
			stats[key] = &stat{
				text:  "",
				value: 0,
			}

			count++
		}
		// end Get current stats from DB

		// Generate our argument list for the statement -> userId, owner, key1, value1, userId, owner, key2, value2, userId, owner, ...
		var args []interface{}
		keys, _ = strconv.Atoi(event.Command.Message["u."+strconv.Itoa(i)+".s.[]"])
		for j := 0; j < keys; j++ {

			if event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".ut"] != "3" {
				log.Noteln("Update new Type:", event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".k"], event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".t"], event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".ut"], event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".v"], event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".pt"])
			}

			key := event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".k"]
			value := event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".t"]

			// Check for forbidden items
			if strings.Contains(value, ";3018;") {
				log.Errorln("Equipped forbidden rocket launcher, skipping " + key)

				answer := make(map[string]string)
				answer["TXN"] = "UpdateStats"
				event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
				fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)
				return
			}

			if value == "" {
				log.Noteln("Updating stat", key+":", event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".v"], "+", stats[key].value)
				// We are dealing with a number
				value = event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".v"]

				// ut seems to be 3 when we need to add up (xp has ut 0 when you level'ed up, otherwise 3)
				if event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".ut"] == "3" {
					intValue, err := strconv.ParseFloat(value, 64)
					if err != nil {
						// Couldn't transfer it to a number, skip updating this stat
						log.Errorln("Skipping stat "+key, err)

						answer := make(map[string]string)
						answer["TXN"] = "UpdateStats"

						event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
						fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)
						return
					}

					if intValue <= 0 || event.Client.RedisState.Get("clientType") == "server" || key == "c_ltp" || key == "c_sln" || key == "c_ltm" || key == "c_slm" || key == "c_wmid0" || key == "c_wmid1" || key == "c_tut" || key == "c_wmid2" {
						// Only allow increasing numbers (like HeroPoints) by the server for now
						newValue := stats[key].value + intValue
						value = strconv.FormatFloat(newValue, 'f', 4, 64)
					} else {
						log.Errorln("Not allowed to process stat", key)
						answer := make(map[string]string)
						answer["TXN"] = "UpdateStats"
						event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
						fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)
						return
					}
				}
			}

			// We need to append 3 values for each insert/update,
			// owner, key and value
			log.Noteln("Updating stats:", userId, owner, key, value)
			args = append(args, userId)
			args = append(args, owner)
			args = append(args, key)
			args = append(args, value)
		}

		_, err = fM.setStatsStatement(keys).Exec(args...)
		if err != nil {
			log.Errorln("Failed setting stats for hero "+owner, err.Error())
		}
	}

	event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)
}
