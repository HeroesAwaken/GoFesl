package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ReviveNetwork/GoFesl/GameSpy"
	"github.com/ReviveNetwork/GoFesl/log"
	"github.com/ReviveNetwork/GoFesl/matchmaking"

	"github.com/HeroesAwaken/GoAwaken/core"
	"github.com/go-redis/redis"
)

// FeslManager - handles incoming and outgoing FESL data
type FeslManager struct {
	name          string
	db            *sql.DB
	redis         *redis.Client
	socket        *GameSpy.SocketTLS
	eventsChannel chan GameSpy.SocketEvent
	batchTicker   *time.Ticker
	stopTicker    chan bool
	server        bool
}

// New creates and starts a new ClientManager
func (fM *FeslManager) New(name string, port string, certFile string, keyFile string, server bool, db *sql.DB, redis *redis.Client) {
	var err error

	fM.socket = new(GameSpy.SocketTLS)
	fM.db = db
	fM.redis = redis
	fM.name = name
	fM.eventsChannel, err = fM.socket.New(fM.name, port, certFile, keyFile)
	fM.stopTicker = make(chan bool, 1)
	fM.server = server

	if err != nil {
		log.Errorln(err)
	}

	go fM.run()
}

func (fM *FeslManager) run() {
	for {
		select {
		case event := <-fM.eventsChannel:
			switch {
			case event.Name == "newClient":
				fM.newClient(event.Data.(GameSpy.EventNewClientTLS))
			case event.Name == "client.command.Hello":
				fM.hello(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.NuLogin":
				fM.NuLogin(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.NuGetPersonas":
				fM.NuGetPersonas(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.NuGetAccount":
				fM.NuGetAccount(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.NuLoginPersona":
				fM.NuLoginPersona(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.GetStatsForOwners":
				fM.GetStatsForOwners(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.GetStats":
				fM.GetStats(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.NuLookupUserInfo":
				fM.NuLookupUserInfo(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.GetPingSites":
				fM.GetPingSites(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.UpdateStats":
				fM.UpdateStats(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.GetTelemetryToken":
				fM.GetTelemetryToken(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.Start":
				fM.Start(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.close":
				fM.close(event.Data.(GameSpy.EventClientTLSClose))
			case event.Name == "client.command":
				fM.LogCommand(event.Data.(GameSpy.EventClientTLSCommand))
				log.Debugf("Got event %s.%s: %v", event.Name, event.Data.(GameSpy.EventClientTLSCommand).Command.Message["TXN"], event.Data.(GameSpy.EventClientTLSCommand).Command)
			default:
				log.Debugf("Got event %s: %v", event.Name, event.Data)
			}
		}
	}
}

// LogCommand - logs detailed FESL command data to a file for further analysis
func (fM *FeslManager) LogCommand(event GameSpy.EventClientTLSCommand) {
	b, err := json.MarshalIndent(event.Command.Message, "", "	")
	if err != nil {
		panic(err)
	}

	commandType := "request"

	os.MkdirAll("./commands/"+event.Command.Query+"."+event.Command.Message["TXN"]+"", 0777)
	err = ioutil.WriteFile("./commands/"+event.Command.Query+"."+event.Command.Message["TXN"]+"/"+commandType, b, 0644)
	if err != nil {
		panic(err)
	}
}

func (fM *FeslManager) logAnswer(msgType string, msgContent map[string]string, msgType2 uint32) {
	b, err := json.MarshalIndent(msgContent, "", "	")
	if err != nil {
		panic(err)
	}

	commandType := "answer"

	os.MkdirAll("./commands/"+msgType+"."+msgContent["TXN"]+"", 0777)
	err = ioutil.WriteFile("./commands/"+msgType+"."+msgContent["TXN"]+"/"+commandType, b, 0644)
	if err != nil {
		panic(err)
	}
}

// GetTelemetryToken - Not being used right now (maybe used in magma more?)
func (fM *FeslManager) GetTelemetryToken(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answer := make(map[string]string)
	answer["TXN"] = "GetTelemetryToken"
	answer["telemetryToken"] = "MTU5LjE1My4yMzUuMjYsOTk0NixlblVTLF7ZmajcnLfGpKSJk53K/4WQj7LRw9asjLHvxLGhgoaMsrDE3bGWhsyb4e6woYKGjJiw4MCBg4bMsrnKibuDppiWxYKditSp0amvhJmStMiMlrHk4IGzhoyYsO7A4dLM26rTgAo%3d"
	answer["enabled"] = "US"
	answer["filters"] = ""
	answer["disabled"] = ""
	event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)
}

// Status - Basic fesl call to get overall service status (called before pnow?)
func (fM *FeslManager) Status(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	log.Noteln("STATUS CALLED")

	answer := make(map[string]string)
	answer["TXN"] = "Status"
	answer["id.id"] = "1"
	answer["id.partition"] = event.Command.Message["partition.partition"]
	answer["sessionState"] = "COMPLETE"
	answer["props.{}.[]"] = "2"
	answer["props.{resultType}"] = "JOIN"

	// Find latest game (do better later)
	gameID := matchmaking.FindAvailableGID()

	answer["props.{games}.0.lid"] = "1"
	answer["props.{games}.0.fit"] = "1001"
	answer["props.{games}.0.gid"] = gameID
	answer["props.{games}.[]"] = "1"
	/*
		answer["props.{games}.1.lid"] = "2"
		answer["props.{games}.1.fit"] = "100"
		answer["props.{games}.1.gid"] = "2"
		answer["props.{games}.1.avgFit"] = "100"
	*/

	event.Client.WriteFESL("pnow", answer, 0x80000000)
	fM.logAnswer("pnow", answer, 0x80000000)
}

// Start - a method of pnow
func (fM *FeslManager) Start(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	log.Noteln("START CALLED")
	log.Noteln(event.Command.Message["partition.partition"])
	answer := make(map[string]string)
	answer["TXN"] = "Start"
	answer["id.id"] = "1"
	answer["id.partition"] = event.Command.Message["partition.partition"]
	event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)

	fM.Status(event)
}

// MysqlRealEscapeString - you know
func MysqlRealEscapeString(value string) string {
	replace := map[string]string{"\\": "\\\\", "'": `\'`, "\\0": "\\\\0", "\n": "\\n", "\r": "\\r", `"`: `\"`, "\x1a": "\\Z"}

	for b, a := range replace {
		value = strings.Replace(value, b, a, -1)
	}

	return value
}

// UpdateStats - updates stats about a soldier
func (fM *FeslManager) UpdateStats(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answer := event.Command.Message
	answer["TXN"] = "UpdateStats"

	users, _ := strconv.Atoi(event.Command.Message["u.[]"])
	for i := 0; i < users; i++ {
		query := ""
		owner, ok := event.Command.Message["u."+strconv.Itoa(i)+".o"]

		if !ok {
			return
		}

		statsNum, _ := strconv.Atoi(event.Command.Message["u."+strconv.Itoa(i)+".s.[]"])
		for j := 0; j < statsNum; j++ {
			if event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".t"] != "" {
				query += event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".k"] + "='" + MysqlRealEscapeString(event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".t"]) + "', "
			} else {
				// TODO: Needs to be fixed, v = change, so v = -1 means substract one
				query += event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".k"] + "='" + MysqlRealEscapeString(event.Command.Message["u."+strconv.Itoa(i)+".s."+strconv.Itoa(j)+".v"]) + "', "
			}
		}

		if owner != "0" && owner != event.Client.RedisState.Get("uID") {
			sql := "UPDATE `awaken_heroes_stats` SET " + query + "pid=" + owner + " WHERE pid = " + owner + ""
			_, err := fM.db.Exec(sql)
			if err != nil {
				log.Errorln(err)
			}
		} else {
			sql := "UPDATE `awaken_heroes_accounts` SET " + query + "uid=" + owner + " WHERE uid = " + owner + ""
			_, err := fM.db.Exec(sql)
			if err != nil {
				log.Errorln(err)
			}
		}
	}

	event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)
}

// GetPingSites - returns a list of endpoints to test for the lowest latency on a client
func (fM *FeslManager) GetPingSites(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answer := make(map[string]string)
	answer["TXN"] = "GetPingSites"
	answer["minPingSitesToPing"] = "0"
	answer["pingSites.[]"] = "4"
	answer["pingSites.0.addr"] = "127.0.0.1"
	answer["pingSites.0.name"] = "gva"
	answer["pingSites.0.type"] = "0"
	answer["pingSites.1.addr"] = "127.0.0.1"
	answer["pingSites.1.name"] = "nrt"
	answer["pingSites.1.type"] = "0"
	answer["pingSites.2.addr"] = "127.0.0.1"
	answer["pingSites.2.name"] = "iad"
	answer["pingSites.2.type"] = "0"
	answer["pingSites.3.addr"] = "127.0.0.1"
	answer["pingSites.3.name"] = "sjc"
	answer["pingSites.3.type"] = "0"

	event.Client.WriteFESL(event.Command.Query, answer, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answer, event.Command.PayloadID)
}

// NuLoginPersona - soldier login command
func (fM *FeslManager) NuLoginPersona(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuLoginPersona"
	loginPacket["lkey"] = event.Client.RedisState.Get("keyHash")
	loginPacket["profileId"] = event.Client.RedisState.Get("uID")
	loginPacket["userId"] = event.Client.RedisState.Get("uID")
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}

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

// NuLookupUserInfo - Gets basic information about a game user
func (fM *FeslManager) NuLookupUserInfo(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	if event.Client.RedisState.Get("clientType") == "server" && event.Command.Message["userInfo.0.userName"] == "gs1-test.revive.systems" {

		log.Noteln("LookupUserInfo - SERVER MODE")
		stmt, err := fM.db.Prepare("SELECT name, id FROM revive_heroes_servers WHERE id =" + event.Client.RedisState.Get("uID"))
		defer stmt.Close()
		if err != nil {
			log.Errorln(err)
			return
		}
		var name, sID string

		err = stmt.QueryRow().Scan(&name, &sID)
		if err != nil {
			log.Errorln(err)
			return
		}

		personaPacket := make(map[string]string)
		personaPacket["TXN"] = "NuLookupUserInfo"
		personaPacket["userInfo.0.userName"] = name
		personaPacket["userInfo.0.userId"] = sID
		personaPacket["userInfo.0.masterUserId"] = sID
		personaPacket["userInfo.0.namespace"] = "MAIN"
		personaPacket["userInfo.0.xuid"] = "158"
		personaPacket["userInfo.0.cid"] = "158"
		//personaPacket["user"] = "1"
		personaPacket["userInfo.[]"] = strconv.Itoa(1)

		event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
		fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)
		return

	}

	log.Noteln("LookupUserInfo - CLIENT MODE! " + event.Command.Message["userInfo.0.userName"])

	userNames := []interface{}{}
	keys, _ := strconv.Atoi(event.Command.Message["userInfo.[]"])
	for i := 0; i < keys; i++ {
		userNames = append(userNames, event.Command.Message["userInfo."+strconv.Itoa(i)+".userName"])
	}

	stmt, err := fM.db.Prepare("SELECT nickname, uid, pid FROM heroes_soldiers WHERE nickname IN (?" + strings.Repeat(",?", len(userNames)-1) + ")")
	defer stmt.Close()
	if err != nil {
		log.Errorln(err)
		return
	}

	rows, err := stmt.Query(userNames...)
	if err != nil {
		log.Errorln(err)
		return
	}

	personaPacket := make(map[string]string)
	personaPacket["TXN"] = "NuLookupUserInfo"
	var k = 0
	for rows.Next() {
		var nickname, webId, pid string
		err := rows.Scan(&nickname, &webId, &pid)
		if err != nil {
			log.Errorln(err)
			return
		}

		personaPacket["userInfo."+strconv.Itoa(k)+".userName"] = nickname
		personaPacket["userInfo."+strconv.Itoa(k)+".userId"] = pid
		personaPacket["userInfo."+strconv.Itoa(k)+".masterUserId"] = pid
		personaPacket["userInfo."+strconv.Itoa(k)+".namespace"] = "MAIN"
		personaPacket["userInfo."+strconv.Itoa(k)+".xuid"] = webId

		k++
	}
	//personaPacket["user"] = "1"
	personaPacket["userInfo.[]"] = strconv.Itoa(k)

	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)

}

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

// NuGetAccount - General account information retrieved, based on parameters sent
func (fM *FeslManager) NuGetAccount(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuGetAccount"
	loginPacket["heroName"] = event.Client.RedisState.Get("username")
	loginPacket["nuid"] = event.Client.RedisState.Get("username") + "@reviveheroes.com"
	loginPacket["DOBDay"] = "1"
	loginPacket["DOBMonth"] = "1"
	loginPacket["DOBYear"] = "2017"
	loginPacket["userId"] = event.Client.RedisState.Get("uID")
	loginPacket["globalOptin"] = "0"
	loginPacket["thidPartyOptin"] = "0"
	loginPacket["language"] = "enUS"
	loginPacket["country"] = "US"
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}

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
	numOfHeroesInt, err := strconv.Atoi(numOfHeroes)
	if err != nil {
		return
	}

	pids := []interface{}{}
	for i := 1; i <= numOfHeroesInt; i++ {
		pids = append(pids, event.Client.RedisState.Get("ownerId."+strconv.Itoa(i)))
	}

	// TODO
	// Check for mysql injection
	var query string
	keys, _ := strconv.Atoi(event.Command.Message["keys.[]"])
	for i := 0; i < keys; i++ {
		query += event.Command.Message["keys."+strconv.Itoa(i)+""] + ", "
	}

	// Result is your slice string.
	rawResult := make([][]byte, keys+1)
	result := make([]string, keys+1)

	dest := make([]interface{}, keys+1) // A temporary interface{} slice
	for i, _ := range rawResult {
		dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
	}

	stmt, err := fM.db.Prepare("SELECT " + query + "pid FROM awaken_heroes_stats WHERE pid IN (?" + strings.Repeat(",?", len(pids)-1) + ")")
	defer stmt.Close()
	if err != nil {
		log.Errorln(err)
		return
	}

	rows, err := stmt.Query(pids...)
	if err != nil {
		log.Errorln(err)
		return
	}

	var k = 0
	for rows.Next() {
		err := rows.Scan(dest...)
		if err != nil {
			log.Errorln(err)
			return
		}

		for i, raw := range rawResult {
			if raw == nil {
				result[i] = "\\N"
			} else {
				result[i] = string(raw)
			}
		}

		keys, _ := strconv.Atoi(event.Command.Message["keys.[]"])

		loginPacket["stats."+strconv.Itoa(k)+".ownerId"] = result[len(result)-1]
		loginPacket["stats."+strconv.Itoa(k)+".ownerType"] = "1"
		loginPacket["stats."+strconv.Itoa(k)+".stats.[]"] = event.Command.Message["keys.[]"]
		for i := 0; i < keys; i++ {
			loginPacket["stats."+strconv.Itoa(k)+".stats."+strconv.Itoa(i)+".key"] = event.Command.Message["keys."+strconv.Itoa(i)+""]
			loginPacket["stats."+strconv.Itoa(k)+".stats."+strconv.Itoa(i)+".value"] = result[i]
			loginPacket["stats."+strconv.Itoa(k)+".stats."+strconv.Itoa(i)+".text"] = result[i]
		}
		k++
	}

	loginPacket["stats.[]"] = strconv.Itoa(k)

	event.Client.WriteFESL(event.Command.Query, loginPacket, 0xC0000007)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}

// GetStats - Get basic stats about a soldier/owner (account holder)
func (fM *FeslManager) GetStats(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "GetStats"

	owner := event.Command.Message["owner"]

	log.Noteln(event.Command.Message["owner"])

	// TODO
	// Check for mysql injection
	var query string
	keys, _ := strconv.Atoi(event.Command.Message["keys.[]"])
	for i := 0; i < keys; i++ {
		query += event.Command.Message["keys."+strconv.Itoa(i)+""] + ", "
	}

	// Result is your slice string.
	rawResult := make([][]byte, keys+1)
	result := make([]string, keys+1)

	dest := make([]interface{}, keys+1) // A temporary interface{} slice
	for i := range rawResult {
		dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
	}

	// Owner==0 is for accounts-stats.
	// Otherwise hero-stats
	if owner == "0" || owner == event.Client.RedisState.Get("uID") {
		stmt, err := fM.db.Prepare("SELECT " + query + "uid FROM awaken_heroes_accounts WHERE uid = ?")
		log.Debugln(stmt)
		defer stmt.Close()
		if err != nil {
			log.Errorln(err)

			// DEV CODE; REMOVE BEFORE TAKING LIVE!!!!!
			// Creates a missing column

			var columns []string
			keys, _ = strconv.Atoi(event.Command.Message["keys.[]"])
			for i := 0; i < keys; i++ {
				columns = append(columns, event.Command.Message["keys."+strconv.Itoa(i)+""])
			}

			for _, column := range columns {
				log.Debugln("Checking column " + column)
				stmt2, err := fM.db.Prepare("SELECT count(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = \"awaken_heroes_accounts\" AND COLUMN_NAME = \"" + column + "\"")
				defer stmt2.Close()
				if err != nil {
				}
				var count int
				err = stmt2.QueryRow().Scan(&count)
				if err != nil {
				}

				if count == 0 {
					log.Debugln("Creating column " + column)
					// If we land here, the column doesn't exist, so create it

					_, err := fM.db.Exec("ALTER TABLE `awaken_heroes_accounts` ADD COLUMN `" + column + "` TEXT NULL")
					if err != nil {
					}
				}
			}

			//DEV CODE; REMOVE BEFORE TAKING LIVE!!!!!
			return
		}

		err = stmt.QueryRow(event.Client.RedisState.Get("uID")).Scan(dest...)
		if err != nil {
			log.Debugln(err)
			return
		}
		if err != nil {
			log.Errorln(err)
			return
		}

		for i, raw := range rawResult {
			if raw == nil {
				result[i] = "\\N"
			} else {
				result[i] = string(raw)
			}
		}

		loginPacket["ownerId"] = result[len(result)-1]
		loginPacket["ownerType"] = "1"
		loginPacket["stats.[]"] = event.Command.Message["keys.[]"]
		for i := 0; i < keys; i++ {
			loginPacket["stats."+strconv.Itoa(i)+".key"] = event.Command.Message["keys."+strconv.Itoa(i)+""]
			loginPacket["stats."+strconv.Itoa(i)+".value"] = result[i]
		}

		event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
		fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)

		return
	}

	// DO the same as above but for hero-stats instead of hero-account

	stmt, err := fM.db.Prepare("SELECT " + query + "pid FROM awaken_heroes_stats WHERE pid = ?")

	defer stmt.Close()
	if err != nil {
		log.Errorln(err)

		// DEV CODE; REMOVE BEFORE TAKING LIVE!!!!!
		// Creates a missing column

		var columns []string
		keys, _ = strconv.Atoi(event.Command.Message["keys.[]"])
		for i := 0; i < keys; i++ {
			columns = append(columns, event.Command.Message["keys."+strconv.Itoa(i)+""])
		}

		for _, column := range columns {
			log.Debugln("Checking column " + column)
			stmt2, err := fM.db.Prepare("SELECT count(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = \"awaken_heroes_stats\" AND COLUMN_NAME = \"" + column + "\"")
			defer stmt2.Close()
			if err != nil {
			}
			var count int
			err = stmt2.QueryRow().Scan(&count)
			if err != nil {
			}

			if count == 0 {
				log.Debugln("Creating column " + column)
				// If we land here, the column doesn't exist, so create it

				sql := "ALTER TABLE `awaken_heroes_stats` ADD COLUMN `" + column + "` TEXT NULL"
				_, err := fM.db.Exec(sql)
				if err != nil {
					log.Errorln(sql)
					log.Errorln(err)
				}
			}
		}

		//DEV CODE; REMOVE BEFORE TAKING LIVE!!!!!
		//return
	}
	log.Noteln(stmt)
	err = stmt.QueryRow(owner).Scan(dest...)
	if err != nil {
		log.Debugln(err)
		return
	}
	if err != nil {
		log.Errorln(err)
		return
	}

	for i, raw := range rawResult {
		if raw == nil {
			result[i] = "\\N"
		} else {
			result[i] = string(raw)
		}
	}

	loginPacket["ownerId"] = result[len(result)-1]
	loginPacket["ownerType"] = "1"
	loginPacket["stats.[]"] = event.Command.Message["keys.[]"]
	for i := 0; i < keys; i++ {
		loginPacket["stats."+strconv.Itoa(i)+".key"] = event.Command.Message["keys."+strconv.Itoa(i)+""]
		loginPacket["stats."+strconv.Itoa(i)+".value"] = result[i]
	}

	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)

}

func (fM *FeslManager) hello(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	redisState := new(core.RedisState)
	redisState.New(fM.redis, event.Command.Message["clientType"]+"-"+event.Client.IpAddr.String())

	event.Client.RedisState = redisState

	if !fM.server {
		getSession := make(map[string]string)
		getSession["TXN"] = "GetSessionId"
		event.Client.WriteFESL("gsum", getSession, 0)
	}

	saveRedis := make(map[string]interface{})
	saveRedis["SDKVersion"] = event.Command.Message["SDKVersion"]
	saveRedis["clientPlatform"] = event.Command.Message["clientPlatform"]
	saveRedis["clientString"] = event.Command.Message["clientString"]
	saveRedis["clientType"] = event.Command.Message["clientType"]
	saveRedis["clientVersion"] = event.Command.Message["clientVersion"]
	saveRedis["locale"] = event.Command.Message["locale"]
	saveRedis["sku"] = event.Command.Message["sku"]
	event.Client.RedisState.SetM(saveRedis)

	helloPacket := make(map[string]string)
	helloPacket["TXN"] = "Hello"
	helloPacket["domainPartition.domain"] = "eagames"
	if fM.server {
		helloPacket["domainPartition.subDomain"] = "bfwest-server"
	} else {
		helloPacket["domainPartition.subDomain"] = "bfwest-dedicated"
	}
	helloPacket["curTime"] = "Jun-15-2017 07:26:12 UTC"
	helloPacket["activityTimeoutSecs"] = "10"
	helloPacket["messengerIp"] = "messaging.ea.com"
	helloPacket["messengerPort"] = "13505"
	helloPacket["theaterIp"] = "alpha.reviveheroes.com"
	if fM.server {
		helloPacket["theaterPort"] = "18056"
	} else {
		helloPacket["theaterPort"] = "18275"
	}
	event.Client.WriteFESL("fsys", helloPacket, 0xC0000001)
	fM.logAnswer("fsys", helloPacket, 0xC0000001)

}

func (fM *FeslManager) newClient(event GameSpy.EventNewClientTLS) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	memCheck := make(map[string]string)
	memCheck["TXN"] = "MemCheck"
	memCheck["memcheck.[]"] = "0"
	memCheck["salt"] = "5"
	event.Client.WriteFESL("fsys", memCheck, 0xC0000000)
	fM.logAnswer("fsys", memCheck, 0xC0000000)

	// Start Heartbeat
	event.Client.State.HeartTicker = time.NewTicker(time.Second * 10)
	go func() {
		for {
			if !event.Client.IsActive {
				return
			}
			select {
			case <-event.Client.State.HeartTicker.C:
				if !event.Client.IsActive {
					return
				}
				memCheck := make(map[string]string)
				memCheck["TXN"] = "MemCheck"
				memCheck["memcheck.[]"] = "0"
				memCheck["salt"] = "5"
				event.Client.WriteFESL("fsys", memCheck, 0xC0000000)
				fM.logAnswer("fsys", memCheck, 0xC0000000)
			}
		}
	}()

	log.Noteln("Client connecting")

}

func (fM *FeslManager) close(event GameSpy.EventClientTLSClose) {
	log.Noteln("Client closed.")

	if event.Client.RedisState != nil {
		event.Client.RedisState.Delete()
	}

	if !event.Client.State.HasLogin {
		return
	}

}

func (fM *FeslManager) error(event GameSpy.EventClientTLSError) {
	log.Noteln("Client threw an error: ", event.Error)
}
