package fesl

import (
	"strconv"

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
		stmt, err := fM.db.Prepare("SELECT id, name, id FROM revive_heroes_servers WHERE secretKey = ?")
		defer stmt.Close()
		if err != nil {
			log.Debugln(err)
			return
		}

		var sID, uID int
		var username string

		err = stmt.QueryRow(event.Command.Message["password"]).Scan(&sID, &username, &uID)
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
		saveRedis["uID"] = strconv.Itoa(uID)
		saveRedis["username"] = username
		saveRedis["apikey"] = event.Command.Message["encryptedInfo"]
		saveRedis["keyHash"] = event.Command.Message["password"]
		event.Client.RedisState.SetM(saveRedis)

		loginPacket := make(map[string]string)
		loginPacket["TXN"] = "NuLogin"
		loginPacket["profileId"] = strconv.Itoa(uID)
		loginPacket["userId"] = strconv.Itoa(uID)
		loginPacket["nuid"] = username
		loginPacket["lkey"] = event.Command.Message["password"]
		event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
		fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
		return
	}

	stmt, err := fM.db.Prepare("SELECT uid, sessionId FROM heroes_beta_members WHERE sessionId = ?")
	defer stmt.Close()
	if err != nil {
		log.Debugln(err)
		return
	}

	var uid int
	var sessionId string
	err = stmt.QueryRow(event.Command.Message["encryptedInfo"]).Scan(&uid, &sessionId)
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

	stmt, err = fM.db.Prepare("SELECT id, username, banned, confirmed_em FROM web_users WHERE id = ?")
	defer stmt.Close()
	if err != nil {
		log.Debugln(err)
		return
	}

	var uID int
	var username string
	var banned, confirmedEm bool

	err = stmt.QueryRow(uid).Scan(&uID, &username, &banned, &confirmedEm)
	if err != nil {
		loginPacket := make(map[string]string)
		loginPacket["TXN"] = "NuLogin"
		loginPacket["localizedMessage"] = "\"Something went wrong\""
		loginPacket["errorContainer.[]"] = "0"
		loginPacket["errorCode"] = "122"
		event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
		return
	}

	// Currently only allow admins & testers
	if !confirmedEm || banned {
		log.Noteln("User not worthy: " + username)
		loginPacket := make(map[string]string)
		loginPacket["TXN"] = "NuLogin"
		loginPacket["localizedMessage"] = "\"Your user information is not confirmed, or you are banned.\""
		loginPacket["errorContainer.[]"] = "0"
		loginPacket["errorCode"] = "120"
		event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
		return
	}

	saveRedis := make(map[string]interface{})
	saveRedis["uID"] = strconv.Itoa(uID)
	saveRedis["username"] = username
	saveRedis["sessionID"] = sessionId
	event.Client.RedisState.SetM(saveRedis)

	// Create a soldier if needed!
	stmt, err = fM.db.Prepare("SELECT COUNT(pid) FROM heroes_soldiers WHERE uid = ?")
	defer stmt.Close()
	if err != nil {
	}

	rows, err := stmt.Query(uid)
	if err != nil {
	}
	var count int
	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			log.Panicln("Wat!?")
		}
	}

	if count == 0 {

		// create awaken_heroes_accounts row
		stmt, err = fM.db.Prepare("INSERT INTO awaken_heroes_accounts (uid) VALUES (?)")
		if err != nil {
		}
		_, err := stmt.Exec(uid)
		if err != nil {
			log.Panicln("Error creating account")
		}

		teamMap := make(map[int]string)
		kitMap := make(map[int]string)

		teamMap[1] = "N"
		teamMap[2] = "R"

		kitMap[0] = "C"
		kitMap[1] = "S"
		kitMap[2] = "G"

		// create soldiers
		for i := 1; i <= 2; i++ {
			team := strconv.Itoa(i)

			for k := 0; k <= 2; k++ {
				// This means we prob need to create a soldier for this fellow
				stmt, err = fM.db.Prepare("INSERT INTO heroes_soldiers (uid, nickname) VALUES (?, ?)")
				if err != nil {
				}
				res, err := stmt.Exec(uid, username+"_"+teamMap[i]+kitMap[k])
				if err != nil {
					log.Panicln("Error creating soldier")
				}

				stmt, err = fM.db.Prepare("INSERT INTO awaken_heroes_stats (pid, c_team, c_kit) VALUES (?, ?, ?)")
				if err != nil {
				}

				//random team for now :)
				lastid, _ := res.LastInsertId()
				_, err = stmt.Exec(lastid, team, strconv.Itoa(k))
				if err != nil {
					log.Panicln("Error creating soldier")
				}
			}

		}

	}
	// End soldier creation

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuLogin"
	loginPacket["profileId"] = strconv.Itoa(uID)
	loginPacket["userId"] = strconv.Itoa(uID)
	loginPacket["nuid"] = username
	loginPacket["lkey"] = "dicks"
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}
