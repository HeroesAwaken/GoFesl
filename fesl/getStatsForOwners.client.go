package fesl

import (
	"strconv"
	"strings"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

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
