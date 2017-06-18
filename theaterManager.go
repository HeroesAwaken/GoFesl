package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	gs "github.com/ReviveNetwork/GoRevive/GameSpy"
	log "github.com/ReviveNetwork/GoRevive/Log"
)

type TheaterManager struct {
	name          string
	socket        *gs.Socket
	eventsChannel chan gs.SocketEvent
	batchTicker   *time.Ticker
	stopTicker    chan bool
}

// New creates and starts a new ClientManager
func (tM *TheaterManager) New(name string, port string) {
	var err error

	tM.socket = new(gs.Socket)
	tM.name = name
	tM.eventsChannel, err = tM.socket.New(tM.name, port, true)
	tM.stopTicker = make(chan bool, 1)

	if err != nil {
		log.Errorln(err)
	}

	go tM.run()
}

func (tM *TheaterManager) run() {
	for {
		select {
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
			case event.Name == "client.command":
				tM.LogCommand(event.Data.(gs.EventClientFESLCommand))
				log.Debugf("Got event %s: %v", event.Name, event.Data.(gs.EventClientFESLCommand).Command)
			default:
				log.Debugf("Got event %s: %v", event.Name, event.Data)
			}
		}
	}
}

func (tM *TheaterManager) EGAM(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["GID"] = event.Command.Message["GID"]
	answerPacket["LID"] = event.Command.Message["LID"]
	event.Client.WriteFESL("EGAM", answerPacket, 0x0)
	tM.logAnswer("EGAM", answerPacket, 0x0)
}

func (tM *TheaterManager) GDAT(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["LID"] = event.Command.Message["LID"]
	answerPacket["GID"] = event.Command.Message["GID"]
	answerPacket["TYPE"] = "G"

	answerPacket["N"] = "hostname"
	answerPacket["I"] = "127.0.0.1"
	answerPacket["P"] = "18567"

	answerPacket["PL"] = "PC"
	answerPacket["V"] = "1.0"

	answerPacket["GN"] = "ServerName"
	answerPacket["HU"] = event.Command.Message["GID"] // ServerID

	answerPacket["J"] = "0"
	answerPacket["JP"] = "0" // joining players?
	answerPacket["AP"] = "0"
	answerPacket["MP"] = "16"

	answerPacket["PW"] = "0"
	answerPacket["QP"] = "0"

	answerPacket["B-version"] = "1.89.239937.0"
	answerPacket["B-numObservers"] = "0"
	answerPacket["B-maxObservers"] = "0"
	answerPacket["B-maxGameSize"] = "16"
	answerPacket["B-U-Character"] = "1"
	answerPacket["B-U-AcceptType"] = "2"
	answerPacket["B-U-FriendlyFire"] = "0"
	answerPacket["B-U-IsDLC"] = "0"
	answerPacket["B-U-UseVoice"] = "0"
	answerPacket["B-U-Duration"] = "2458"
	answerPacket["B-U-Map"] = "village"
	answerPacket["B-U-DlcMapId"] = "0"
	answerPacket["B-U-Mission"] = "PmcCon001"
	answerPacket["B-U-Money"] = "181000"
	answerPacket["B-U-Oil"] = "125"
	event.Client.WriteFESL(event.Command.Query, answerPacket, 0x0)
	tM.logAnswer(event.Command.Query, answerPacket, 0x0)

	answerPacket = make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["LID"] = event.Command.Message["LID"]
	answerPacket["GID"] = event.Command.Message["GID"]
	answerPacket["GUID"] = ""
	event.Client.WriteFESL("GDET", answerPacket, 0x0)
	tM.logAnswer("GDET", answerPacket, 0x0)

	answerPacket = make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["LID"] = event.Command.Message["LID"]
	answerPacket["GID"] = event.Command.Message["GID"]
	answerPacket["NAME"] = "ServerName"
	answerPacket["UID"] = event.Command.Message["GID"]
	answerPacket["PID"] = "1"
	event.Client.WriteFESL("PDAT", answerPacket, 0x0)
	tM.logAnswer("PDAT", answerPacket, 0x0)

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
