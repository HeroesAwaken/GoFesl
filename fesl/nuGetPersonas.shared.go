package fesl

import (
	"strconv"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// NuGetPersonas - Soldier data lookup call
func (fM *FeslManager) NuGetPersonas(event GameSpy.EventClientTLSCommand) {

	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	if event.Client.RedisState.Get("clientType") == "server" {
		log.Noteln("We are a server NuGetPersonas")
		// Server login
		stmt, err := fM.db.Prepare("SELECT name, id FROM revive_heroes_servers WHERE id = ?")
		log.Noteln(stmt)
		defer stmt.Close()
		if err != nil {
			return
		}

		rows, err := stmt.Query(event.Client.RedisState.Get("uID"))
		if err != nil {
			return
		}

		personaPacket := make(map[string]string)
		personaPacket["TXN"] = "NuGetPersonas"

		var i = 0
		for rows.Next() {
			var name string
			var id int
			err := rows.Scan(&name, &id)
			if err != nil {
				log.Errorln(err)
				return
			}
			personaPacket["personas."+strconv.Itoa(i)] = name
			event.Client.RedisState.Set("ownerId."+strconv.Itoa(i+1), strconv.Itoa(id))
			i++
		}

		personaPacket["personas.[]"] = strconv.Itoa(i)

		event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
		fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)
		log.Noteln(event.Command.Query, personaPacket, event.Command.PayloadID)
		return
	}

	stmt, err := fM.db.Prepare("SELECT nickname, pid FROM heroes_soldiers WHERE uid = ?")
	log.Noteln(stmt)
	defer stmt.Close()
	if err != nil {
		return
	}

	rows, err := stmt.Query(event.Client.RedisState.Get("uID"))
	if err != nil {
		return
	}

	personaPacket := make(map[string]string)
	personaPacket["TXN"] = "NuGetPersonas"

	var i = 0
	for rows.Next() {
		var username string
		var pid int
		err := rows.Scan(&username, &pid)
		if err != nil {
			log.Errorln(err)
			return
		}
		personaPacket["personas."+strconv.Itoa(i)] = username
		event.Client.RedisState.Set("ownerId."+strconv.Itoa(i+1), strconv.Itoa(pid))
		i++
	}

	event.Client.RedisState.Set("numOfHeroes", strconv.Itoa(i))

	personaPacket["personas.[]"] = strconv.Itoa(i)

	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)
}
