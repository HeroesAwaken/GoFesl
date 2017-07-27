package fesl

import (
	"strconv"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// GetStatsForOwners - Gives a bunch of info for the Hero selection screen?
func (fM *FeslManager) GetStatsForOwners(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "GetStats"

	// Get the owner pids from redis
	numOfHeroes := event.Client.RedisState.Get("numOfHeroes")
	userID := event.Client.RedisState.Get("uID")
	numOfHeroesInt, err := strconv.Atoi(numOfHeroes)
	if err != nil {
		return
	}

	i := 1
	for i = 1; i <= numOfHeroesInt; i++ {
		ownerID := event.Client.RedisState.Get("ownerId." + strconv.Itoa(i))

		loginPacket["stats."+strconv.Itoa(i-1)+".ownerId"] = ownerID
		loginPacket["stats."+strconv.Itoa(i-1)+".ownerType"] = "1"

		// Generate our argument list for the statement -> heroID, key1, key2, key3, ...
		var args []interface{}
		statsKeys := make(map[string]string)
		args = append(args, ownerID)
		args = append(args, userID)
		keys, _ := strconv.Atoi(event.Command.Message["keys.[]"])
		for i := 0; i < keys; i++ {
			args = append(args, event.Command.Message["keys."+strconv.Itoa(i)+""])
			statsKeys[event.Command.Message["keys."+strconv.Itoa(i)+""]] = strconv.Itoa(i)
		}

		rows, err := fM.getStatsStatement(keys).Query(args...)
		if err != nil {
			log.Errorln("Failed gettings stats for hero "+ownerID, err.Error())
		}

		count := 0
		for rows.Next() {
			var userID, heroID, statsKey, statsValue string
			err := rows.Scan(&userID, &heroID, &statsKey, &statsValue)
			if err != nil {
				log.Errorln("Issue with database:", err.Error())
			}

			loginPacket["stats."+strconv.Itoa(i-1)+".stats."+strconv.Itoa(count)+".key"] = statsKey
			loginPacket["stats."+strconv.Itoa(i-1)+".stats."+strconv.Itoa(count)+".value"] = statsValue
			loginPacket["stats."+strconv.Itoa(i-1)+".stats."+strconv.Itoa(count)+".text"] = statsValue

			delete(statsKeys, statsKey)
			count++
		}

		// Send stats not found with default value of ""
		for key := range statsKeys {
			loginPacket["stats."+strconv.Itoa(i-1)+".stats."+strconv.Itoa(count)+".key"] = key
			loginPacket["stats."+strconv.Itoa(i-1)+".stats."+strconv.Itoa(count)+".value"] = "0"
			loginPacket["stats."+strconv.Itoa(i-1)+".stats."+strconv.Itoa(count)+".text"] = ""

			count++
		}
		loginPacket["stats."+strconv.Itoa(i-1)+".stats.[]"] = strconv.Itoa(count)
	}

	loginPacket["stats.[]"] = strconv.Itoa(i - 1)

	event.Client.WriteFESL(event.Command.Query, loginPacket, 0xC0000007)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}
