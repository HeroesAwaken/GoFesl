package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	gs "github.com/ReviveNetwork/GoRevive/GameSpy"
	log "github.com/ReviveNetwork/GoRevive/Log"
	"github.com/ReviveNetwork/GoRevive/core"
	"github.com/go-redis/redis"
)

type GameServer struct {
	ip                 string
	port               string
	intIP              string
	intPort            string
	name               string
	level              string
	activePlayers      int
	maxPlayers         int
	queueLength        int
	joiningPlayers     int
	gameMode           string
	elo                float64
	numObservers       int
	maxObservers       int
	sguid              string
	hash               string
	password           string
	ugid               string
	sType              string
	join               string
	version            string
	dataCenter         string
	serverMap          string
	armyBalance        string
	armyDistribution   string
	availSlotsNational bool
	availSlotsRoyal    bool
	avgAllyRank        float64
	avgAxisRank        float64
	serverState        string
	communityName      string
}

type TheaterManager struct {
	name             string
	socket           *gs.Socket
	socketUDP        *gs.SocketUDP
	db               *sql.DB
	redis            *redis.Client
	eventsChannel    chan gs.SocketEvent
	eventsChannelUDP chan gs.SocketUDPEvent
	batchTicker      *time.Ticker
	stopTicker       chan bool
	gameServerGlobal *core.RedisState
}

// New creates and starts a new ClientManager
func (tM *TheaterManager) New(name string, port string, db *sql.DB, redis *redis.Client) {
	var err error

	tM.socket = new(gs.Socket)
	tM.socketUDP = new(gs.SocketUDP)
	tM.db = db
	tM.redis = redis
	tM.name = name
	tM.eventsChannel, err = tM.socket.New(tM.name, port, true)
	if err != nil {
		log.Errorln(err)
	}
	tM.eventsChannelUDP, err = tM.socketUDP.New(tM.name, port, true)
	if err != nil {
		log.Errorln(err)
	}
	tM.stopTicker = make(chan bool, 1)

	tM.gameServerGlobal = new(core.RedisState)
	tM.gameServerGlobal.New(tM.redis, "gameServer-config")
	tM.gameServerGlobal.Set("Lobbies", "0")

	go tM.run()
}

func (tM *TheaterManager) run() {
	for {
		select {
		case event := <-tM.eventsChannelUDP:
			switch {
			case event.Name == "command.ECHO":
				go tM.ECHO(event)
			case event.Name == "command":
				tM.LogCommandUDP(event.Data.(*gs.CommandFESL))
				log.Debugf("UDP Got event %s: %v", event.Name, event.Data.(*gs.CommandFESL))
			default:
				log.Debugf("UDP Got event %s: %v", event.Name, event.Data)
			}
		case event := <-tM.eventsChannel:
			switch {
			case event.Name == "newClient":
				go tM.newClient(event.Data.(gs.EventNewClient))
			case event.Name == "client.command.CONN":
				go tM.CONN(event.Data.(gs.EventClientFESLCommand))
			case event.Name == "client.command.USER":
				go tM.USER(event.Data.(gs.EventClientFESLCommand))
			case event.Name == "client.command.LLST":
				go tM.LLST(event.Data.(gs.EventClientFESLCommand))
			case event.Name == "client.command.GDAT":
				go tM.GDAT(event.Data.(gs.EventClientFESLCommand))
			case event.Name == "client.command.EGAM":
				go tM.EGAM(event.Data.(gs.EventClientFESLCommand))
			case event.Name == "client.command.ECNL":
				go tM.ECNL(event.Data.(gs.EventClientFESLCommand))
			case event.Name == "client.command.CGAM":
				go tM.CGAM(event.Data.(gs.EventClientFESLCommand))
			case event.Name == "client.command.UBRA":
				go tM.UBRA(event.Data.(gs.EventClientFESLCommand))
			case event.Name == "client.command.UGAM":
				go tM.UGAM(event.Data.(gs.EventClientFESLCommand))
			case event.Name == "client.command":
				tM.LogCommand(event.Data.(gs.EventClientFESLCommand))
				log.Debugf("Got event %s: %v", event.Name, event.Data.(gs.EventClientFESLCommand).Command)
			default:
				log.Debugf("Got event %s: %v", event.Name, event.Data)
			}
		}
	}
}

func (tM *TheaterManager) ECHO(event gs.SocketUDPEvent) {
	command := event.Data.(*gs.CommandFESL)

	answerPacket := make(map[string]string)
	answerPacket["TID"] = command.Message["TID"]
	answerPacket["TXN"] = command.Message["TXN"]
	answerPacket["IP"] = event.Addr.IP.String()
	answerPacket["PORT"] = strconv.Itoa(event.Addr.Port)
	answerPacket["ERR"] = "0"
	answerPacket["TYPE"] = "1"
	err := tM.socketUDP.WriteFESL("ECHO", answerPacket, 0x0, event.Addr)
	if err != nil {
		log.Errorln(err)
	}
	tM.logAnswer("ECHO", answerPacket, 0x0)
}

func (tM *TheaterManager) ECNL(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["GID"] = event.Command.Message["GID"]
	answerPacket["LID"] = event.Command.Message["LID"]
	event.Client.WriteFESL("ECNL", answerPacket, 0x0)
	tM.logAnswer("ECNL", answerPacket, 0x0)
}

func (tM *TheaterManager) EGAM(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	serverFull := false

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["GID"] = event.Command.Message["GID"]
	answerPacket["LID"] = event.Command.Message["LID"]

	gameServer := new(core.RedisState)
	gameServer.New(tM.redis, "gameServer-"+event.Command.Message["LID"])

	maxPlayers, _ := strconv.Atoi(gameServer.Get("MAX-PLAYERS"))
	tmpPlayers, _ := strconv.Atoi(gameServer.Get("ACTIVE-PLAYERS"))

	if tmpPlayers+1 > maxPlayers {
		// Server is full
		log.Noteln("Server full")
		tmpPlayers, _ := strconv.Atoi(gameServer.Get("QUEUE-LENGTH"))
		gameServer.Set("QUEUE-LENGTH", strconv.Itoa(tmpPlayers))
		tmpPlayers++
		serverFull = true
	}

	if !serverFull {
		event.Client.WriteFESL("EGAM", answerPacket, 0x0)
		tM.logAnswer("EGAM", answerPacket, 0x0)
	}

	event.Client.WriteFESL("EGAM", answerPacket, 0x0)
	tM.logAnswer("EGAM", answerPacket, 0x0)
}

func (tM *TheaterManager) CGAM(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	addr, ok := event.Client.IpAddr.(*net.TCPAddr)

	if !ok {
		log.Errorln("Failed turning IpAddr to net.TCPAddr")
		return
	}
	/*
		gameServer := GameServer{}
		gameServer.ip = addr.IP.String()
		gameServer.port = event.Command.Message["PORT"]
		gameServer.intIP = event.Command.Message["INT-IP"]
		gameServer.intPort = event.Command.Message["INT-PORT"]
		gameServer.name = event.Command.Message["NAME"]
		gameServer.level = event.Command.Message["B-U-map"]
		gameServer.activePlayers = 0
		gameServer.maxPlayers, _ = strconv.Atoi(event.Command.Message["MAX-PLAYERS"])
		gameServer.queueLength = 0
		gameServer.joiningPlayers = 0
		gameServer.gameMode = ""
		gameServer.elo, _ = strconv.ParseFloat(event.Command.Message["B-U-elo_rank"], 64)
		gameServer.numObservers, _ = strconv.Atoi(event.Command.Message["B-numObservers"])
		gameServer.maxObservers, _ = strconv.Atoi(event.Command.Message["B-maxObservers"])
		gameServer.sguid = ""
		gameServer.hash = ""
		gameServer.password = ""
		gameServer.ugid = event.Command.Message["UGID"]
		gameServer.sType = event.Command.Message["TYPE"]
		gameServer.join = event.Command.Message["JOIN"]
		gameServer.version = event.Command.Message["B-version"]
		gameServer.dataCenter = event.Command.Message["B-U-data_center"]
		gameServer.serverMap = event.Command.Message["B-U-map"]
		gameServer.armyBalance = event.Command.Message["B-U-army_balance"]
		gameServer.armyDistribution = event.Command.Message["B-U-army_distribution"]
		gameServer.availSlotsNational, _ = strconv.ParseBool(event.Command.Message["B-U-avail_slots_national"])
		gameServer.availSlotsRoyal, _ = strconv.ParseBool(event.Command.Message["B-U-avail_slots_royal"])
		gameServer.avgAllyRank, _ = strconv.ParseFloat(event.Command.Message["B-U-avg_ally_rank"], 64)
		gameServer.avgAxisRank, _ = strconv.ParseFloat(event.Command.Message["B-U-avg_axis_rank"], 64)
		gameServer.serverState = event.Command.Message["B-U-server_state"]
		gameServer.communityName = event.Command.Message["B-U-community_name"]
	*/

	currentLobbyId := tM.gameServerGlobal.Get("Lobbies")
	gameLid, _ := strconv.Atoi(currentLobbyId)
	gameLid++

	gameServer := new(core.RedisState)
	gameServer.New(tM.redis, "gameServer-"+strconv.Itoa(gameLid))

	for index, value := range event.Command.Message {
		gameServer.Set(index, value)
	}

	gameServer.Set("LID", strconv.Itoa(gameLid))
	gameServer.Set("IP", addr.IP.String())
	gameServer.Set("ACTIVE-PLAYERS", "0")
	gameServer.Set("QUEUE-LENGTH", "0")

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["MAX-PLAYERS"] = "16"
	answerPacket["EKEY"] = "AIBSgPFqRDg0TfdXW1zUGa4%3d"
	answerPacket["UGID"] = event.Command.Message["UGID"]
	answerPacket["JOIN"] = event.Command.Message["JOIN"]
	answerPacket["LID"] = strconv.Itoa(gameLid)
	answerPacket["SECRET"] = "4l94N6Y0A3Il3+kb55pVfK6xRjc+Z6sGNuztPeNGwN5CMwC7ZlE/lwel07yciyZ5y3bav7whbzHugPm11NfuBg%3d%3d"
	answerPacket["J"] = event.Command.Message["JOIN"]
	answerPacket["GID"] = "1"
	event.Client.WriteFESL("CGAM", answerPacket, 0x0)
	tM.logAnswer("CGAM", answerPacket, 0x0)

	tM.gameServerGlobal.Set("Lobbies", strconv.Itoa(gameLid))
}

func (tM *TheaterManager) GDAT(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	gameServer := new(core.RedisState)
	gameServer.New(tM.redis, "gameServer-"+event.Command.Message["LID"])

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["LID"] = event.Command.Message["LID"]
	answerPacket["GID"] = event.Command.Message["GID"]

	answerPacket["HU"] = "bfwest.server.p"
	answerPacket["HN"] = "1"

	answerPacket["I"] = gameServer.Get("IP")
	answerPacket["P"] = gameServer.Get("PORT")
	answerPacket["N"] = gameServer.Get("NAME")
	answerPacket["AP"] = gameServer.Get("ACTIVE-PLAYERS")
	answerPacket["MP"] = gameServer.Get("MAX-PLAYERS")
	answerPacket["QP"] = "0"
	answerPacket["JP"] = "0"
	answerPacket["PL"] = "PC"

	answerPacket["PW"] = "0"
	answerPacket["TYPE"] = gameServer.Get("TYPE")
	answerPacket["J"] = gameServer.Get("JOIN")

	for _, key := range gameServer.HKeys() {
		if strings.Index(key, "B-") != -1 {
			answerPacket[key] = gameServer.Get(key)
		}
	}

	/*
		answerPacket["UGID"] = gameServer.Get("UGID")
		answerPacket["HTTYPE"] = gameServer.Get("HTTYPE")
		answerPacket["HXFR"] = gameServer.Get("HXFR")
		answerPacket["RT"] = gameServer.Get("RT")
		answerPacket["IP"] = gameServer.Get("IP")
		answerPacket["PORT"] = gameServer.Get("PORT")
		answerPacket["RESERVE-HOST"] = gameServer.Get("RESERVE-HOST")
		answerPacket["QLEN"] = "0"
		answerPacket["SECRET"] = gameServer.Get("SECRET")
		answerPacket["JOIN"] = gameServer.Get("JOIN")
		answerPacket["DISABLE-AUTO-DEQUEUE"] = gameServer.Get("DISABLE-AUTO-DEQUEUE")
		answerPacket["NAME"] = gameServer.Get("NAME")
		answerPacket["INT-IP"] = gameServer.Get("INT-IP")
		answerPacket["INT-PORT"] = gameServer.Get("INT-PORT")
		answerPacket["ACTIVE-PLAYERS"] = gameServer.Get("ACTIVE-PLAYERS")
		answerPacket["QUEUE-LENGTH"] = gameServer.Get("QUEUE-LENGTH")
		answerPacket["MAX-PLAYERS"] = gameServer.Get("MAX-PLAYERS")
	*/

	answerPacket["B-version"] = "1.89.239937.0"

	event.Client.WriteFESL("GDAT", answerPacket, 0x0)
	tM.logAnswer("GDAT", answerPacket, 0x0)

	time.Sleep(100 * time.Millisecond)

	answerPacket = make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["LID"] = event.Command.Message["LID"]
	answerPacket["GID"] = event.Command.Message["GID"]
	answerPacket["NAME"] = "ServerName"
	answerPacket["UID"] = event.Command.Message["GID"]
	answerPacket["PID"] = "1"
	event.Client.WriteFESL("PDAT", answerPacket, 0x0)
	tM.logAnswer("PDAT", answerPacket, 0x0)

	answerPacket = make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["LID"] = event.Command.Message["LID"]
	answerPacket["GID"] = event.Command.Message["GID"]
	answerPacket["UGID"] = gameServer.Get("UGID")

	answerPacket["D-AutoBalance"] = ""
	answerPacket["D-Crosshair"] = ""
	answerPacket["D-FriendlyFire"] = ""
	answerPacket["D-KillCam"] = ""
	answerPacket["D-Minimap"] = ""
	answerPacket["D-MinimapSpotting"] = ""

	answerPacket["D-ServerDescriptionCount"] = "0"

	answerPacket["D-ThirdPersonVehicleCameras"] = ""
	answerPacket["D-ThreeDSpotting"] = ""

	maxPlayers, _ := strconv.Atoi(gameServer.Get("MAX-PLAYERS"))

	for i := 0; i < maxPlayers; i++ {
		entry := gameServer.Get("PDAT00" + strconv.Itoa(i))
		if entry != "|0|0|0|0" && entry != "" {
		} else {
			entry = "|0|0|0|0"
		}

		key := "D-pdat" + strconv.Itoa(i)
		if i < 10 {
			key = "D-pdat0" + strconv.Itoa(i)
		}
		answerPacket[key] = entry
	}

	event.Client.WriteFESL("GDET", answerPacket, 0x0)
	tM.logAnswer("GDET", answerPacket, 0x0)

}

func (tM *TheaterManager) LogCommandUDP(event *gs.CommandFESL) {
	b, err := json.MarshalIndent(event.Message, "", "	")
	if err != nil {
		panic(err)
	}

	commandType := "request"

	os.MkdirAll("./commands/"+event.Query+"."+event.Message["TXN"]+"", 0777)
	err = ioutil.WriteFile("./commands/"+event.Query+"."+event.Message["TXN"]+"/"+commandType, b, 0644)
	if err != nil {
		panic(err)
	}
}

func (tM *TheaterManager) LogCommand(event gs.EventClientFESLCommand) {
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

func (tM *TheaterManager) logAnswer(msgType string, msgContent map[string]string, msgType2 uint32) {
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

func (tM *TheaterManager) LLST(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["NUM-LOBBIES"] = "1"
	event.Client.WriteFESL(event.Command.Query, answerPacket, 0x0)

	ldatPacket := make(map[string]string)
	ldatPacket["TID"] = "LDAT"
	ldatPacket["FAVORITE-GAMES"] = "0"
	ldatPacket["FAVORITE-PLAYERS"] = "0"
	ldatPacket["LID"] = "257"
	ldatPacket["LOCALE"] = "en_US"
	ldatPacket["MAX-GAMES"] = "10000"
	ldatPacket["NAME"] = "bfheroesPC1"
	ldatPacket["NUM-GAMES"] = "7"
	ldatPacket["PASSING"] = "7"
	event.Client.WriteFESL("LDAT", ldatPacket, 0x0)
	tM.logAnswer("LDAT", ldatPacket, 0x0)
}

func (tM *TheaterManager) USER(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["NAME"] = "MakaHost"
	answerPacket["CID"] = "1"
	event.Client.WriteFESL(event.Command.Query, answerPacket, 0x0)
	tM.logAnswer(event.Command.Query, answerPacket, 0x0)
}

func (tM *TheaterManager) UBRA(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	event.Client.WriteFESL(event.Command.Query, answerPacket, 0x0)
	tM.logAnswer(event.Command.Query, answerPacket, 0x0)
}

func (tM *TheaterManager) UGAM(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	gameServer := new(core.RedisState)
	gameServer.New(tM.redis, "gameServer-"+event.Command.Message["LID"])

	log.Noteln("Updating GameServer " + event.Command.Message["LID"])

	for index, value := range event.Command.Message {
		log.Noteln("SET " + index + " " + value)
		gameServer.Set(index, value)
	}
}

func (tM *TheaterManager) CONN(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["TIME"] = strconv.FormatInt(time.Now().UTC().Unix(), 10)
	answerPacket["activityTimeoutSecs"] = "15"
	answerPacket["PROT"] = event.Command.Message["PROT"]
	event.Client.WriteFESL(event.Command.Query, answerPacket, 0x0)
	tM.logAnswer(event.Command.Query, answerPacket, 0x0)
}

func (tM *TheaterManager) newClient(event gs.EventNewClient) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	log.Noteln("Client connecting")

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
				pingPacket := make(map[string]string)
				pingPacket["TID"] = "0"
				event.Client.WriteFESL("PING", pingPacket, 0x0)
			}
		}
	}()
}

func (tM *TheaterManager) close(event gs.EventClientTLSClose) {
	log.Noteln("Client closed.")

	if !event.Client.State.HasLogin {
		return
	}

}

func (tM *TheaterManager) error(event gs.EventClientTLSError) {
	log.Noteln("Client threw an error: ", event.Error)
}
