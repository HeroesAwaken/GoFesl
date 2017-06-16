package main

import (
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
			case event.Name == "client.command":
				log.Debugf("Got event %s: %v", event.Name, event.Data.(gs.EventClientFESLCommand).Command)
			default:
				log.Debugf("Got event %s: %v", event.Name, event.Data)
			}
		}
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
}

func (tM *TheaterManager) USER(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := event.Command.Message
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["NAME"] = "MakaHost"
	answerPacket["CID"] = "3"
	event.Client.WriteFESL(event.Command.Query, answerPacket, 0x0)
}

func (tM *TheaterManager) CONN(event gs.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TID"] = event.Command.Message["TID"]
	answerPacket["TIME"] = strconv.FormatInt(time.Now().UTC().Unix(), 10)
	answerPacket["activityTimeoutSecs"] = "5"
	answerPacket["PROT"] = "2"
	event.Client.WriteFESL(event.Command.Query, answerPacket, 0x0)
}

func (tM *TheaterManager) newClient(event gs.EventNewClient) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	log.Noteln("Client connecting")

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
