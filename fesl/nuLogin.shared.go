package fesl

import (
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// NuLogin - master login command
func (fM *FeslManager) NuLogin(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	if event.Client.RedisState.Get("clientType") == "server" {
		// Server login

		var id, userID, servername, secretKey, username string

		err := fM.stmtGetServerBySecret.QueryRow(event.Command.Message["password"]).Scan(&id, &userID, &servername, &secretKey, &username)
		if err != nil {
			loginPacket := make(map[string]string)
			loginPacket["TXN"] = "NuLogin"
			loginPacket["localizedMessage"] = "\"The password the user specified is incorrect\""
			loginPacket["errorContainer.[]"] = "0"
			loginPacket["errorCode"] = "122"
			event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
			return
		}

		saveRedis := make(map[string]interface{})
		saveRedis["uID"] = userID
		saveRedis["username"] = username
		saveRedis["apikey"] = event.Command.Message["encryptedInfo"]
		saveRedis["keyHash"] = event.Command.Message["password"]
		event.Client.RedisState.SetM(saveRedis)

		loginPacket := make(map[string]string)
		loginPacket["TXN"] = "NuLogin"
		loginPacket["profileId"] = userID
		loginPacket["userId"] = userID
		loginPacket["nuid"] = username
		loginPacket["lkey"] = event.Command.Message["password"]
		event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
		fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
		return
	}

	var id, username, email, birthday, language, country, gameToken string

	err := fM.stmtGetUserByGameToken.QueryRow(event.Command.Message["encryptedInfo"]).Scan(&id, &username, &email, &birthday, &language, &country, &gameToken)
	if err != nil {
		log.Noteln("User not worthy!")
		loginPacket := make(map[string]string)
		loginPacket["TXN"] = "NuLogin"
		loginPacket["localizedMessage"] = "\"The user is not entitled to access this game\""
		loginPacket["errorContainer.[]"] = "0"
		loginPacket["errorCode"] = "120"
		event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
		return
	}

	// Check if user is allowed to login
	if !fM.userHasPermission(id, "game.login") {
		log.Noteln("User not worthy: " + username)
		loginPacket := make(map[string]string)
		loginPacket["TXN"] = "NuLogin"
		loginPacket["localizedMessage"] = "\"Your user is currently not allowed to login.\""
		loginPacket["errorContainer.[]"] = "0"
		loginPacket["errorCode"] = "120"
		event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
		return
	}

	saveRedis := make(map[string]interface{})
	saveRedis["uID"] = id
	saveRedis["username"] = username
	saveRedis["sessionID"] = gameToken
	event.Client.RedisState.SetM(saveRedis)

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuLogin"
	loginPacket["profileId"] = id
	loginPacket["userId"] = id
	loginPacket["nuid"] = username
	loginPacket["lkey"] = "dicks"
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}
