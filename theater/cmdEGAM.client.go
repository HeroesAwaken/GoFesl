package theater

import (
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/lib"
	"github.com/SpencerSharkey/GoFesl/log"
	"github.com/SpencerSharkey/GoFesl/matchmaking"
)

// EGAM - CLIENT called when a client wants to join a gameserver
func (tM *TheaterManager) EGAM(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	//externalIP := event.Client.IpAddr.(*net.TCPAddr).IP.String()
	lobbyID := event.Command.Message["LID"]
	gameID := event.Command.Message["GID"]
	pid := event.Client.RedisState.Get("id")

	clientAnswer := make(map[string]string)
	clientAnswer["TID"] = event.Command.Message["TID"]
	clientAnswer["LID"] = lobbyID
	clientAnswer["GID"] = gameID
	event.Client.WriteFESL("EGAM", clientAnswer, 0x0)
	tM.logAnswer("EGAM", clientAnswer, 0x0)

	// Get 4 stats for PID
	rows, err := tM.getStatsStatement(4).Query(pid, "c_kit", "c_team", "elo", "level")
	if err != nil {
		log.Errorln("Failed gettings stats for hero "+pid, err.Error())
	}

	stats := make(map[string]string)

	for rows.Next() {
		var userID, heroID, heroName, statsKey, statsValue string
		err := rows.Scan(&userID, &heroID, &heroName, &statsKey, &statsValue)
		if err != nil {
			log.Errorln("Issue with database:", err.Error())
		}

		stats["heroName"] = heroName
		stats["userID"] = userID
		stats[statsKey] = statsValue
	}

	// todo: get game data and check if full

	if gameServer, ok := matchmaking.Games[gameID]; ok {
		gsData := new(lib.RedisObject)
		gsData.New(tM.redis, "gdata", gameID)

		//gameServer := matchmaking.Games[gameID]

		serverEGRQ := make(map[string]string)
		serverEGRQ["TID"] = event.Command.Message["TID"]

		serverEGRQ["NAME"] = stats["heroName"]
		serverEGRQ["UID"] = event.Command.Message["R-U-accid"]
		//serverEGRQ["PID"] = event.Command.Message["R-U-accid"]
		serverEGRQ["PID"] = pid
		serverEGRQ["TICKET"] = "2018751182"
		serverEGRQ["cid"] = pid

		serverEGRQ["IP"] = event.Command.Message["R-U-externalIp"]
		//serverEGRQ["IP"] = externalIP
		//serverEGRQ["PORT"] = strconv.Itoa(event.Client.IpAddr.(*net.TCPAddr).Port)
		serverEGRQ["PORT"] = event.Command.Message["PORT"]

		serverEGRQ["INT-IP"] = event.Command.Message["R-INT-IP"]
		serverEGRQ["INT-PORT"] = event.Command.Message["R-INT-PORT"]

		serverEGRQ["PTYPE"] = "P"
		// maybe do CID here?
		serverEGRQ["R-USER"] = stats["heroName"]
		serverEGRQ["R-UID"] = event.Command.Message["R-U-accid"]
		serverEGRQ["R-U-accid"] = event.Command.Message["R-U-accid"]
		serverEGRQ["R-U-elo"] = stats["elo"]
		serverEGRQ["R-U-team"] = stats["c_team"]
		serverEGRQ["R-U-kit"] = stats["c_kit"]
		serverEGRQ["R-U-lvl"] = stats["level"]
		serverEGRQ["R-U-dataCenter"] = "iad"
		serverEGRQ["R-U-externalIp"] = event.Command.Message["R-U-externalIp"]
		//serverEGRQ["R-U-externalIp"] = externalIP
		serverEGRQ["R-U-internalIp"] = event.Command.Message["R-INT-IP"]
		serverEGRQ["R-U-category"] = event.Command.Message["R-U-category"]
		serverEGRQ["R-U-cid"] = pid
		serverEGRQ["R-INT-IP"] = event.Command.Message["R-INT-IP"]
		serverEGRQ["R-INT-PORT"] = event.Command.Message["R-INT-PORT"]

		serverEGRQ["XUID"] = event.Command.Message["R-U-accid"]
		serverEGRQ["R-XUID"] = event.Command.Message["R-U-accid"]
		serverEGRQ["R-cid"] = pid

		serverEGRQ["LID"] = lobbyID
		serverEGRQ["GID"] = gameID

		gameServer.WriteFESL("EGRQ", serverEGRQ, 0x0)
		tM.logAnswer("EGRQ", serverEGRQ, 0x0)

		clientEGEG := make(map[string]string)
		clientEGEG["TID"] = event.Command.Message["TID"]
		clientEGEG["PL"] = "pc"
		clientEGEG["TICKET"] = "2018751182"

		// That is the ServerID, was/is a test
		clientEGEG["PID"] = "3"
		clientEGEG["I"] = gsData.Get("IP")
		clientEGEG["P"] = gsData.Get("PORT")
		clientEGEG["HUID"] = "1" // find via GID soon
		clientEGEG["EKEY"] = "O65zZ2D2A58mNrZw1hmuJw%3d%3d"
		clientEGEG["INT-IP"] = gsData.Get("INT-IP")
		clientEGEG["INT-PORT"] = gsData.Get("INT-PORT")
		clientEGEG["SECRET"] = "2587913"
		clientEGEG["UGID"] = gsData.Get("UGID")
		clientEGEG["LID"] = lobbyID
		clientEGEG["GID"] = gameID

		event.Client.WriteFESL("EGEG", clientEGEG, 0x0)
		tM.logAnswer("EGEG", clientEGEG, 0x0)
	}

}
