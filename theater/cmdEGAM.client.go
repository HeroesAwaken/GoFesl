package theater

import (
	"net"
	"strconv"

	"github.com/ReviveNetwork/GoFesl/GameSpy"
	"github.com/ReviveNetwork/GoFesl/lib"
	"github.com/ReviveNetwork/GoFesl/log"
	"github.com/ReviveNetwork/GoFesl/matchmaking"
)

// EGAM - CLIENT called when a client wants to join a gameserver
func (tM *TheaterManager) EGAM(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	externalIP := event.Client.IpAddr.(*net.TCPAddr).IP.String()
	lobbyID := event.Command.Message["LID"]
	gameID := event.Command.Message["GID"]
	pid := event.Command.Message["R-U-accid"]

	clientAnswer := make(map[string]string)
	clientAnswer["TID"] = event.Command.Message["TID"]
	clientAnswer["LID"] = lobbyID
	clientAnswer["GID"] = gameID
	event.Client.WriteFESL("EGAM", clientAnswer, 0x0)
	tM.logAnswer("EGAM", clientAnswer, 0x0)

	// Get client data
	stmt, err := tM.db.Prepare("SELECT uid, nickname FROM heroes_soldiers WHERE pid = ?")
	defer stmt.Close()
	if err != nil {
		log.Errorln(err)
		return
	}
	var nickname string
	var uid int
	stmt.QueryRow(pid).Scan(&uid, &nickname)

	// Get statas data
	statsStmt, err := tM.db.Prepare("SELECT c_kit, c_team, elo, level FROM awaken_heroes_stats WHERE pid = ?")
	defer stmt.Close()
	if err != nil {
		log.Errorln(err)
		return
	}
	var pKit, pTeam, pElo, pLevel string
	statsStmt.QueryRow(pid).Scan(&pKit, &pTeam, &pElo, &pLevel)

	// todo: get game data and check if full

	if gameServer, ok := matchmaking.Games[gameID]; ok {
		gsData := new(lib.RedisObject)
		gsData.New(tM.redis, "gdata", gameID)

		//gameServer := matchmaking.Games[gameID]

		serverEGRQ := make(map[string]string)
		serverEGRQ["TID"] = "0"

		serverEGRQ["NAME"] = nickname
		serverEGRQ["UID"] = strconv.Itoa(uid)
		serverEGRQ["PID"] = pid
		serverEGRQ["TICKET"] = "2018751182"

		serverEGRQ["IP"] = externalIP
		serverEGRQ["PORT"] = strconv.Itoa(event.Client.IpAddr.(*net.TCPAddr).Port)

		serverEGRQ["INT-IP"] = event.Command.Message["R-INT-IP"]
		serverEGRQ["INT-PORT"] = event.Command.Message["R-INT-PORT"]

		serverEGRQ["PTYPE"] = "P"
		// maybe do CID here?
		serverEGRQ["R-USER"] = nickname
		serverEGRQ["R-UID"] = strconv.Itoa(uid)
		serverEGRQ["R-U-accid"] = strconv.Itoa(uid)
		serverEGRQ["R-U-elo"] = pElo
		serverEGRQ["R-U-team"] = pTeam
		serverEGRQ["R-U-kit"] = pKit
		serverEGRQ["R-U-lvl"] = pLevel
		serverEGRQ["R-U-dataCenter"] = "iad"
		serverEGRQ["R-U-externalIp"] = externalIP
		serverEGRQ["R-U-internalIp"] = event.Command.Message["R-INT-IP"]
		serverEGRQ["R-U-category"] = event.Command.Message["R-U-category"]
		serverEGRQ["R-INT-IP"] = event.Command.Message["R-INT-IP"]
		serverEGRQ["R-INT-PORT"] = event.Command.Message["R-INT-PORT"]

		serverEGRQ["XUID"] = "24"
		serverEGRQ["R-XUID"] = "24"

		serverEGRQ["LID"] = lobbyID
		serverEGRQ["GID"] = gameID

		gameServer.WriteFESL("EGRQ", serverEGRQ, 0x0)
		tM.logAnswer("EGRQ", serverEGRQ, 0x0)

		clientEGEG := make(map[string]string)
		clientEGEG["TID"] = event.Command.Message["TID"]
		clientEGEG["PL"] = "pc"
		clientEGEG["TICKET"] = "2018751182"
		clientEGEG["PID"] = pid
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
