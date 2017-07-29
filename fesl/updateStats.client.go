package fesl

import (
	"strconv"

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

			key := event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".k"]
			value := event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".t"]

			if value == "" {
				// We are dealing with a number
				value = event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".v"]
				intValue, err := strconv.ParseFloat(value, 64)
				if err != nil {
					// Couldn't transfer it to a number, skip updating this stat
					log.Noteln("Skipping stat " + key)
					continue
				}

				if intValue <= 0 || event.Client.RedisState.Get("clientType") != "server" {
					// Only allow increasing numbers (like HeroPoints) by the server for now
					value = strconv.FormatFloat(stats[key].value+intValue, 'f', 4, 64)
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
