package fesl

import (
	"strconv"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// GetStats - Get basic stats about a soldier/owner (account holder)
func (fM *FeslManager) GetStats(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	owner := event.Command.Message["owner"]
	log.Noteln(event.Command.Message["owner"])

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "GetStats"
	loginPacket["ownerId"] = owner
	loginPacket["ownerType"] = "1"

	// Generate our argument list for the statement -> heroID, key1, key2, key3, ...
	var args []interface{}
	args = append(args, owner)
	keys, _ := strconv.Atoi(event.Command.Message["keys.[]"])
	for i := 0; i < keys; i++ {
		args = append(args, event.Command.Message["keys."+strconv.Itoa(i)+""])
	}

	rows, err := fM.getStatsStatement(keys).Query(args...)
	if err != nil {
		log.Errorln("Failed gettings stats for hero "+owner, err.Error())
	}

	count := 0
	for rows.Next() {
		var heroID, key, value string
		err := rows.Scan(&heroID, &key, &value)
		if err != nil {
			log.Errorln("Issue with database:", err.Error())
		}

		loginPacket["stats."+strconv.Itoa(count)+".key"] = key
		loginPacket["stats."+strconv.Itoa(count)+".value"] = value
		count++
	}
	loginPacket["stats.[]"] = strconv.Itoa(count)

	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)

}
