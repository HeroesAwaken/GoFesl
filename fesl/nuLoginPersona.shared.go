package fesl

import (
	"github.com/HeroesAwaken/GoFesl2/lib"
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// NuLoginPersona - soldier login command
func (fM *FeslManager) NuLoginPersona(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	if event.Client.RedisState.Get("clientType") == "server" {
		// Server login
		fM.NuLoginPersonaServer(event)
		return
	}

	var id, userID, heroName, online string
	err := fM.stmtGetHeroeByName.QueryRow(event.Command.Message["name"]).Scan(&id, &userID, &heroName, &online)
	if err != nil {
		log.Noteln("Persona not worthy!")
		return
	}

	// Setup a new key for our persona
	lkey := GameSpy.BF2RandomUnsafe(24)
	lkeyRedis := new(lib.RedisObject)
	lkeyRedis.New(fM.redis, "lkeys", lkey)
	lkeyRedis.Set("id", id)
	lkeyRedis.Set("userID", userID)
	lkeyRedis.Set("name", heroName)

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuLoginPersona"
	loginPacket["lkey"] = lkey
	loginPacket["profileId"] = id
	loginPacket["userId"] = id
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}

// NuLoginPersonaServer - soldier login command
func (fM *FeslManager) NuLoginPersonaServer(event GameSpy.EventClientTLSCommand) {
	var id, userID, servername, secretKey, username string
	err := fM.stmtGetServerByName.QueryRow(event.Command.Message["name"]).Scan(&id, &userID, &servername, &secretKey, &username)
	if err != nil {
		log.Noteln("Persona not worthy!")
		return
	}

	// Setup a new key for our persona
	lkey := GameSpy.BF2RandomUnsafe(24)
	lkeyRedis := new(lib.RedisObject)
	lkeyRedis.New(fM.redis, "lkeys", lkey)
	lkeyRedis.Set("id", id)
	lkeyRedis.Set("userID", userID)
	lkeyRedis.Set("name", servername)

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuLoginPersona"
	loginPacket["lkey"] = lkey
	loginPacket["profileId"] = id
	loginPacket["userId"] = id
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}
