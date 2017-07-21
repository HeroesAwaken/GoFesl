package fesl

import (
	"strconv"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

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
