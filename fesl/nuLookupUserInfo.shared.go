package fesl

import (
	"strconv"
	"strings"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

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
