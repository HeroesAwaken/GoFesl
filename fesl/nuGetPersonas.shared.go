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
		fM.NuGetPersonasServer(event)
		return
	}

	rows, err := fM.stmtGetHeroesByUserID.Query(event.Client.RedisState.Get("uID"))
	if err != nil {
		return
	}

	personaPacket := make(map[string]string)
	personaPacket["TXN"] = "NuGetPersonas"

	var i = 0
	for rows.Next() {
		var id, userID, heroName, online string
		err := rows.Scan(&id, &userID, &heroName, &online)
		if err != nil {
			log.Errorln(err)
			return
		}
		personaPacket["personas."+strconv.Itoa(i)] = heroName
		event.Client.RedisState.Set("ownerId."+strconv.Itoa(i+1), id)
		i++
	}

	event.Client.RedisState.Set("numOfHeroes", strconv.Itoa(i))

	personaPacket["personas.[]"] = strconv.Itoa(i)

	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)
}

// NuGetPersonasServer - Soldier data lookup call for servers
func (fM *FeslManager) NuGetPersonasServer(event GameSpy.EventClientTLSCommand) {
	log.Noteln("We are a server NuGetPersonas")

	// Server login
	rows, err := fM.stmtGetServerByID.Query(event.Client.RedisState.Get("uID"))
	if err != nil {
		return
	}

	personaPacket := make(map[string]string)
	personaPacket["TXN"] = "NuGetPersonas"

	var i = 0
	for rows.Next() {
		var id, userID, servername, secretKey, username string
		err := rows.Scan(&id, &userID, &servername, &secretKey, &username)
		if err != nil {
			log.Errorln(err)
			return
		}
		personaPacket["personas."+strconv.Itoa(i)] = servername
		event.Client.RedisState.Set("ownerId."+strconv.Itoa(i+1), id)
		i++
	}

	personaPacket["personas.[]"] = strconv.Itoa(i)

	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)
	log.Noteln(event.Command.Query, personaPacket, event.Command.PayloadID)
}
