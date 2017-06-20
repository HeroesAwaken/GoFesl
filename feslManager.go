package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	gs "github.com/ReviveNetwork/GoRevive/GameSpy"
	log "github.com/ReviveNetwork/GoRevive/Log"
	"github.com/ReviveNetwork/GoRevive/core"
	"github.com/go-redis/redis"
)

type FeslManager struct {
	name          string
	db            *sql.DB
	redis         *redis.Client
	socket        *gs.SocketTLS
	eventsChannel chan gs.SocketEvent
	batchTicker   *time.Ticker
	stopTicker    chan bool
	server        bool
}

// New creates and starts a new ClientManager
func (fM *FeslManager) New(name string, port string, certFile string, keyFile string, server bool, db *sql.DB, redis *redis.Client) {
	var err error

	fM.socket = new(gs.SocketTLS)
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
				fM.newClient(event.Data.(gs.EventNewClientTLS))
			case event.Name == "client.command.Hello":
				fM.hello(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuLogin":
				fM.NuLogin(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuGetPersonas":
				fM.NuGetPersonas(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuGetAccount":
				fM.NuGetAccount(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuLoginPersona":
				fM.NuLoginPersona(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.GetStatsForOwners":
				fM.GetStatsForOwners(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.GetStats":
				fM.GetStats(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuLookupUserInfo":
				fM.NuLookupUserInfo(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.GetPingSites":
				fM.GetPingSites(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.UpdateStats":
				fM.UpdateStats(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.GetTelemetryToken":
				fM.GetTelemetryToken(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.Start":
				fM.Start(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.close":
				fM.close(event.Data.(gs.EventClientTLSClose))
			case event.Name == "client.command":
				fM.LogCommand(event.Data.(gs.EventClientTLSCommand))
				log.Debugf("Got event %s.%s: %v", event.Name, event.Data.(gs.EventClientTLSCommand).Command.Message["TXN"], event.Data.(gs.EventClientTLSCommand).Command)
			default:
				log.Debugf("Got event %s: %v", event.Name, event.Data)
			}
		}
	}
}

func (fM *FeslManager) LogCommand(event gs.EventClientTLSCommand) {
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

// Not being used right now
func (fM *FeslManager) GetTelemetryToken(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TXN"] = "GetTelemetryToken"
	answerPacket["telemetryToken"] = "MTU5LjE1My4yMzUuMjYsOTk0NixlblVTLF7ZmajcnLfGpKSJk53K/4WQj7LRw9asjLHvxLGhgoaMsrDE3bGWhsyb4e6woYKGjJiw4MCBg4bMsrnKibuDppiWxYKditSp0amvhJmStMiMlrHk4IGzhoyYsO7A4dLM26rTgAo%3d"
	answerPacket["enabled"] = "US"
	answerPacket["filters"] = ""
	answerPacket["disabled"] = ""
	event.Client.WriteFESL(event.Command.Query, answerPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answerPacket, event.Command.PayloadID)
}

func (fM *FeslManager) Status(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TXN"] = "Status"
	answerPacket["id.id"] = "605"
	answerPacket["id.partition"] = event.Command.Message["partition.partition"]
	answerPacket["sessionState"] = "COMPLETE"
	answerPacket["props.{}"] = "3"
	answerPacket["props.{resultType}"] = "JOIN"
	answerPacket["props.{availableServerCount}"] = "1"
	answerPacket["props.{avgFit}"] = "100"

	answerPacket["props.{games}.[]"] = "1"
	answerPacket["props.{games}.0.lid"] = "1"
	answerPacket["props.{games}.0.gid"] = "1"
	answerPacket["props.{games}.0.fit"] = "1"
	answerPacket["props.{games}.0.avgFit"] = "1"
	/*
		answerPacket["props.{games}.1.lid"] = "2"
		answerPacket["props.{games}.1.fit"] = "100"
		answerPacket["props.{games}.1.gid"] = "2"
		answerPacket["props.{games}.1.avgFit"] = "100"
	*/

	event.Client.WriteFESL("pnow", answerPacket, 0x80000000)
	fM.logAnswer("pnow", answerPacket, 0x80000000)
}

func (fM *FeslManager) Start(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TXN"] = "Start"
	answerPacket["id.id"] = "605"
	answerPacket["id.partition"] = event.Command.Message["partition.partition"]
	event.Client.WriteFESL(event.Command.Query, answerPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answerPacket, event.Command.PayloadID)

	fM.Status(event)
}

func MysqlRealEscapeString(value string) string {
	replace := map[string]string{"\\": "\\\\", "'": `\'`, "\\0": "\\\\0", "\n": "\\n", "\r": "\\r", `"`: `\"`, "\x1a": "\\Z"}

	for b, a := range replace {
		value = strings.Replace(value, b, a, -1)
	}

	return value
}

func (fM *FeslManager) UpdateStats(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := event.Command.Message
	answerPacket["TXN"] = "UpdateStats"

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
			sql := "UPDATE `revive_heroes_stats` SET " + query + "pid=" + owner + " WHERE pid = " + owner + ""
			_, err := fM.db.Exec(sql)
			if err != nil {
				log.Errorln(err)
			}
		} else {
			sql := "UPDATE `revive_heroes_accounts` SET " + query + "uid=" + owner + " WHERE uid = " + owner + ""
			_, err := fM.db.Exec(sql)
			if err != nil {
				log.Errorln(err)
			}
		}
	}

	event.Client.WriteFESL(event.Command.Query, answerPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answerPacket, event.Command.PayloadID)
}

func (fM *FeslManager) GetPingSites(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TXN"] = "GetPingSites"
	answerPacket["minPingSitesToPing"] = "0"
	answerPacket["pingSites.[]"] = "4"
	answerPacket["pingSites.0.addr"] = "127.0.0.1"
	answerPacket["pingSites.0.name"] = "gva"
	answerPacket["pingSites.0.type"] = "0"
	answerPacket["pingSites.1.addr"] = "127.0.0.1"
	answerPacket["pingSites.1.name"] = "nrt"
	answerPacket["pingSites.1.type"] = "0"
	answerPacket["pingSites.2.addr"] = "127.0.0.1"
	answerPacket["pingSites.2.name"] = "iad"
	answerPacket["pingSites.2.type"] = "0"
	answerPacket["pingSites.3.addr"] = "127.0.0.1"
	answerPacket["pingSites.3.name"] = "sjc"
	answerPacket["pingSites.3.type"] = "0"

	event.Client.WriteFESL(event.Command.Query, answerPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answerPacket, event.Command.PayloadID)
}

func (fM *FeslManager) NuLoginPersona(event gs.EventClientTLSCommand) {
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

// Done with redis CLIENT
func (fM *FeslManager) NuLogin(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	if event.Client.RedisState.Get("clientType") == "server" {
		// Server login
		stmt, err := fM.db.Prepare("SELECT t1.id, t2.username, t2.id  FROM revive_heroes_servers t1 LEFT JOIN web_users t2 ON t1.uid=t2.id WHERE t1.secretKey = ?")
		defer stmt.Close()
		if err != nil {
			log.Debugln(err)
			return
		}

		var sID, uID int
		var username string

		err = stmt.QueryRow(event.Command.Message["encryptedInfo"]).Scan(&sID, &username, &uID)
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
		event.Client.RedisState.SetM(saveRedis)

		loginPacket := make(map[string]string)
		loginPacket["TXN"] = "NuLogin"
		loginPacket["profileId"] = strconv.Itoa(uID)
		loginPacket["userId"] = strconv.Itoa(uID)
		loginPacket["nuid"] = username
		loginPacket["lkey"] = event.Command.Message["encryptedInfo"]
		event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
		fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
		return
	}

	stmt, err := fM.db.Prepare("SELECT t1.uid, t1.sessionid, t1.ip, t2.username, t2.banned, t2.is_admin, t2.is_tester, t2.confirmed_em, t2.key_hash, t2.email, t2.country FROM web_sessions t1 LEFT JOIN web_users t2 ON t1.uid=t2.id WHERE t1.sessionid = ?")
	defer stmt.Close()
	if err != nil {
		log.Debugln(err)
		return
	}

	var uID int
	var ip, username, sessionID, keyHash, email, country string
	var banned, isAdmin, isTester, confirmedEm bool

	err = stmt.QueryRow(event.Command.Message["encryptedInfo"]).Scan(&uID, &sessionID, &ip, &username, &banned, &isAdmin, &isTester, &confirmedEm, &keyHash, &email, &country)
	if err != nil {
		loginPacket := make(map[string]string)
		loginPacket["TXN"] = "NuLogin"
		loginPacket["localizedMessage"] = "\"The password the user specified is incorrect\""
		loginPacket["errorContainer.[]"] = "0"
		loginPacket["errorCode"] = "122"
		event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
		return
	}

	// Currently only allow admins & testers
	if sessionID != event.Command.Message["encryptedInfo"] || !confirmedEm || banned || (!isAdmin && !isTester) {
		log.Noteln("User not worthy: " + username)
		loginPacket := make(map[string]string)
		loginPacket["TXN"] = "NuLogin"
		loginPacket["localizedMessage"] = "\"The user is not entitled to access this game\""
		loginPacket["errorContainer.[]"] = "0"
		loginPacket["errorCode"] = "120"
		event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
		return
	}

	saveRedis := make(map[string]interface{})
	saveRedis["uID"] = strconv.Itoa(uID)
	saveRedis["username"] = username
	saveRedis["ip"] = ip
	saveRedis["sessionID"] = sessionID
	saveRedis["keyHash"] = keyHash
	saveRedis["email"] = email
	saveRedis["country"] = country
	event.Client.RedisState.SetM(saveRedis)

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuLogin"
	loginPacket["profileId"] = strconv.Itoa(uID)
	loginPacket["userId"] = strconv.Itoa(uID)
	loginPacket["nuid"] = username
	loginPacket["lkey"] = keyHash
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}

func (fM *FeslManager) NuLookupUserInfo(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	userNames := []interface{}{}
	keys, _ := strconv.Atoi(event.Command.Message["userInfo.[]"])
	for i := 0; i < keys; i++ {
		userNames = append(userNames, event.Command.Message["userInfo."+strconv.Itoa(i)+".userName"])
	}

	stmt, err := fM.db.Prepare("SELECT nickname, web_id, pid FROM revive_soldiers WHERE nickname IN (?" + strings.Repeat(",?", len(userNames)-1) + ")")
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
			return
		}

		personaPacket["userInfo."+strconv.Itoa(k)+".userName"] = nickname
		personaPacket["userInfo."+strconv.Itoa(k)+".userId"] = pid
		personaPacket["userInfo."+strconv.Itoa(k)+".masterUserId"] = webId
		personaPacket["userInfo."+strconv.Itoa(k)+".namespace"] = "MAIN"

		k++
	}
	//personaPacket["user"] = "1"
	personaPacket["userInfo.[]"] = strconv.Itoa(k)

	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)
}

func (fM *FeslManager) NuGetPersonas(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	stmt, err := fM.db.Prepare("SELECT nickname, pid FROM revive_soldiers WHERE web_id = ? AND game = ?")
	defer stmt.Close()
	if err != nil {
		return
	}

	rows, err := stmt.Query(event.Client.RedisState.Get("uID"), "heroes")
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

func (fM *FeslManager) NuGetAccount(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuGetAccount"
	loginPacket["heroName"] = event.Client.RedisState.Get("username")
	loginPacket["nuid"] = event.Client.RedisState.Get("email")
	loginPacket["DOBDay"] = "1"
	loginPacket["DOBMonthg"] = "1"
	loginPacket["DOBYear"] = "2017"
	loginPacket["userId"] = event.Client.RedisState.Get("uID")
	loginPacket["globalOptin"] = "0"
	loginPacket["thidPartyOptin"] = "0"
	loginPacket["language"] = "en"
	loginPacket["country"] = event.Client.RedisState.Get("country")
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}

func (fM *FeslManager) GetStatsForOwners(event gs.EventClientTLSCommand) {
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

	stmt, err := fM.db.Prepare("SELECT " + query + "pid FROM revive_heroes_stats WHERE pid IN (?" + strings.Repeat(",?", len(pids)-1) + ")")
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

func (fM *FeslManager) GetStats(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "GetStats"

	owner := event.Command.Message["owner"]

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

	// Owner==0 is for accounts-stats.
	// Otherwise hero-stats
	if owner == "0" || owner == event.Client.RedisState.Get("uID") {
		stmt, err := fM.db.Prepare("SELECT " + query + "uid FROM revive_heroes_accounts WHERE uid = ?")
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
				stmt2, err := fM.db.Prepare("SELECT count(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = \"revive_heroes_accounts\" AND COLUMN_NAME = \"" + column + "\"")
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

					_, err := fM.db.Exec("ALTER TABLE `revive_heroes_accounts` ADD COLUMN `" + column + "` TEXT NULL")
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

	stmt, err := fM.db.Prepare("SELECT " + query + "pid FROM revive_heroes_stats WHERE pid = ?")

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
			stmt2, err := fM.db.Prepare("SELECT count(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = \"revive_heroes_stats\" AND COLUMN_NAME = \"" + column + "\"")
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

				sql := "ALTER TABLE `revive_heroes_stats` ADD COLUMN `" + column + "` TEXT NULL"
				_, err := fM.db.Exec(sql)
				if err != nil {
					log.Errorln(sql)
					log.Errorln(err)
				}
			}
		}

		//DEV CODE; REMOVE BEFORE TAKING LIVE!!!!!
		return
	}

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

func (fM *FeslManager) hello(event gs.EventClientTLSCommand) {
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
	helloPacket["activityTimeoutSecs"] = "0"
	helloPacket["messengerIp"] = "messaging.ea.com"
	helloPacket["messengerPort"] = "13505"
	helloPacket["theaterIp"] = "bfwest-server.theater.ea.com"
	if fM.server {
		helloPacket["theaterPort"] = "18056"
	} else {
		helloPacket["theaterPort"] = "18275"
	}
	event.Client.WriteFESL("fsys", helloPacket, 0xC0000001)
	fM.logAnswer("fsys", helloPacket, 0xC0000001)

}

func (fM *FeslManager) newClient(event gs.EventNewClientTLS) {
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
				pingSite := make(map[string]string)
				pingSite["TXN"] = "Ping"
				event.Client.WriteFESL("fsys", pingSite, 0)
			}
		}
	}()

	log.Noteln("Client connecting")

}

func (fM *FeslManager) close(event gs.EventClientTLSClose) {
	log.Noteln("Client closed.")

	if event.Client.RedisState != nil {
		event.Client.RedisState.Delete()
	}

	if !event.Client.State.HasLogin {
		return
	}

}

func (fM *FeslManager) error(event gs.EventClientTLSError) {
	log.Noteln("Client threw an error: ", event.Error)
}
