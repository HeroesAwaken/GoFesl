package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"time"

	gs "github.com/HeroesAwaken/GoAwaken/GameSpy"
	log "github.com/HeroesAwaken/GoAwaken/Log"
	"github.com/HeroesAwaken/GoAwaken/core"
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

	currentLobbyId := tM.gameServerGlobal.Get("Lobbies")
	gameLid, _ := strconv.Atoi(currentLobbyId)
	gameLid++

	gameServer := new(core.RedisState)
	gameServer.New(tM.redis, "gameServer-"+strconv.Itoa(gameLid))

	for index, value := range event.Command.Message {
		// Strip quotes
		if len(value) > 0 && value[0] == '"' {
			value = value[1:]
		}
		if len(value) > 0 && value[len(value)-1] == '"' {
			value = value[:len(value)-1]
		}
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

	answerPacket["LID"] = "1"
	answerPacket["GID"] = "1"
	answerPacket["TYPE"] = "G"
	answerPacket["HN"] = "MakaHost"
	answerPacket["HU"] = "1"
	answerPacket["N"] = "Test"
	answerPacket["I"] = "127.0.0.1"
	answerPacket["P"] = "18567"
	answerPacket["J"] = "0"
	answerPacket["JP"] = "0"
	answerPacket["QP"] = "0"
	answerPacket["AP"] = "0"
	answerPacket["MP"] = "16"
	answerPacket["F"] = ""
	answerPacket["NF"] = ""
	answerPacket["PL"] = "PC"
	answerPacket["PW"] = ""
	answerPacket["B-U-EA"] = "1"
	answerPacket["B-U-Softcore"] = "0"
	answerPacket["B-U-Hardcore"] = "0"
	answerPacket["B-U-HasPassword"] = "0"
	answerPacket["B-U-Punkbuster"] = "0"
	answerPacket["B-version"] = "0"
	answerPacket["V"] = "0"
	answerPacket["B-U-level"] = "village"
	answerPacket["B-U-gamemode"] = "gpm_tdm"
	answerPacket["B-U-sguid"] = ""
	answerPacket["B-U-Time="] = "10"
	answerPacket["B-U-hash"] = ""
	answerPacket["B-U-region"] = ""
	answerPacket["B-U-public"] = "0"
	answerPacket["B-U-elo"] = "0"
	answerPacket["B-numObservers"] = "0"
	answerPacket["B-maxObservers"] = "0"
	answerPacket["B-U-Provider"] = ""
	answerPacket["B-U-gameMod"] = "bfheroes"
	answerPacket["B-U-QueueLength"] = "0"

	event.Client.WriteFESL("GDAT", answerPacket, 0x0)
	tM.logAnswer("GDAT", answerPacket, 0x0)

	/*
		answerPacket["B-maxObservers"] = "0"
		answerPacket["B-numObservers"] = "0"
		answerPacket["B-U-alwaysQueue"] = "1"
		answerPacket["B-U-army_balance"] = "Balanced"
		answerPacket["B-U-army_distribution"] = "0,0,0,0,0,0,0,0,0,0,0"
		answerPacket["B-U-avail_slots_national"] = "yes"
		answerPacket["B-U-avail_slots_royal"] = "yes"
		answerPacket["B-U-avail_vips_national"] = "4"
		answerPacket["B-U-avail_vips_royal"] = "4"
		answerPacket["B-U-avg_ally_rank"] = "1000"
		answerPacket["B-U-avg_axis_rank"] = "1000"
		answerPacket["B-U-community_name"] = "Heroes SV"
		answerPacket["B-U-data_center"] = "iad"
		answerPacket["B-U-easyzone"] = "no"
		answerPacket["B-U-elo_rank"] = "1000"
		answerPacket["B-U-lvl_avg"] = "0.000000"
		answerPacket["B-U-lvl_sdv"] = "0.000000"
		answerPacket["B-U-map"] = "village"
		answerPacket["B-U-map_name"] = "Village"
		answerPacket["B-U-percent_full"] = "0"
		answerPacket["B-U-punkb"] = "0"
		answerPacket["B-U-ranked"] = "yes"
		answerPacket["B-U-server_ip"] = "127.0.0.1"
		answerPacket["B-U-server_port"] = "18567"
		answerPacket["B-U-server_state"] = "empty"
		answerPacket["B-U-servertype"] = "public"
		answerPacket["B-version"] = "1.89.239937.0"

		answerPacket["GID"] = "1"
		answerPacket["I"] = "127.0.0.1"
		answerPacket["J"] = "O"
		answerPacket["LID"] = "1"
		answerPacket["MP"] = "16"
		answerPacket["N"] = "[iad]A Battlefield Heroes Server(127.0.0.1:18567)"
		answerPacket["P"] = "18567"
		answerPacket["PL"] = "PC"
		answerPacket["TID"] = event.Command.Message["TID"]
		answerPacket["TYPE"] = "G"
	*/
	/*
			answerPacket["TID"] = event.Command.Message["TID"]
			answerPacket["LID"] = event.Command.Message["LID"]
			answerPacket["GID"] = event.Command.Message["GID"]

			answerPacket["HU"] = "bfwest-pc"
			answerPacket["HN"] = "1"

			answerPacket["I"] = "127.0.0.1"
			answerPacket["P"] = gameServer.Get("PORT")
			answerPacket["N"] = gameServer.Get("NAME")
			answerPacket["AP"] = gameServer.Get("ACTIVE-PLAYERS")
			answerPacket["MP"] = gameServer.Get("MAX-PLAYERS")
			answerPacket["QP"] = gameServer.Get("QUEUE-LENGTH")
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

			answerPacket["B-version"] = "1.89.239937.0"
			answerPacket["V"] = "1.89.239937.0"

		answerPacket["TID"] = event.Command.Message["TID"]
		event.Client.WriteFESL("GDAT", answerPacket, 0x0)
		tM.logAnswer("GDAT", answerPacket, 0x0)
	*/

	answerPacket = make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["LID"] = event.Command.Message["LID"]
	answerPacket["GID"] = event.Command.Message["GID"]
	answerPacket["D-AutoBalance"] = "1"
	answerPacket["D-Crosshair"] = "1"
	answerPacket["D-FriendlyFire"] = "1"
	answerPacket["D-KillCam"] = "1"
	answerPacket["D-Minimap"] = "1"
	answerPacket["D-MinimapSpotting"] = "1"
	answerPacket["D-ServerDescriptionCount"] = "0"

	answerPacket["D-ThirdPersonVehicleCameras"] = "0"
	answerPacket["D-ThreeDSpotting"] = "0"
	answerPacket["UGID"] = "1"

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
		// Strip quotes
		if len(value) > 0 && value[0] == '"' {
			value = value[1:]
		}
		if len(value) > 0 && value[len(value)-1] == '"' {
			value = value[:len(value)-1]
		}
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
