package fesl

import (
	"strconv"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// NuLookupUserInfo - Gets basic information about a game user
func (fM *FeslManager) NuLookupUserInfo(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	if event.Client.RedisState.Get("clientType") == "server" && event.Command.Message["userInfo.0.userName"] == "Test-Server" {
		fM.NuLookupUserInfoServer(event)
		return
	}

	log.Noteln("LookupUserInfo - CLIENT MODE! " + event.Command.Message["userInfo.0.userName"])

	personaPacket := make(map[string]string)
	personaPacket["TXN"] = "NuLookupUserInfo"

	keys, _ := strconv.Atoi(event.Command.Message["userInfo.[]"])
	for i := 0; i < keys; i++ {
		heroNamePacket := event.Command.Message["userInfo."+strconv.Itoa(i)+".userName"]

		var id, userID, heroName, online string
		err := fM.stmtGetHeroeByName.QueryRow(heroNamePacket).Scan(&id, &userID, &heroName, &online)
		if err != nil {
			return
		}

		personaPacket["userInfo."+strconv.Itoa(i)+".userName"] = heroName
		personaPacket["userInfo."+strconv.Itoa(i)+".userId"] = id
		personaPacket["userInfo."+strconv.Itoa(i)+".masterUserId"] = id
		personaPacket["userInfo."+strconv.Itoa(i)+".namespace"] = "MAIN"
		personaPacket["userInfo."+strconv.Itoa(i)+".xuid"] = "24"
	}

	personaPacket["userInfo.[]"] = strconv.Itoa(keys)

	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)

}

// NuLookupUserInfoServer - Gets basic information about a game user
func (fM *FeslManager) NuLookupUserInfoServer(event GameSpy.EventClientTLSCommand) {
	var err error

	var id, userID, servername, secretKey, username string
	err = fM.stmtGetServerByID.QueryRow(event.Client.RedisState.Get("sID")).Scan(&id, &userID, &servername, &secretKey, &username)
	if err != nil {
		log.Errorln(err)
		return
	}

	personaPacket := make(map[string]string)
	personaPacket["TXN"] = "NuLookupUserInfo"
	personaPacket["userInfo.0.userName"] = servername
	personaPacket["userInfo.0.userId"] = "1"
	personaPacket["userInfo.0.masterUserId"] = "1"
	personaPacket["userInfo.0.namespace"] = "MAIN"
	personaPacket["userInfo.0.xuid"] = "24"
	personaPacket["userInfo.0.cid"] = "1"
	personaPacket["userInfo.[]"] = strconv.Itoa(1)

	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)
}
