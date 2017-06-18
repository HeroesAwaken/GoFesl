package main

import (
	"time"

	gs "github.com/ReviveNetwork/GoRevive/GameSpy"
	log "github.com/ReviveNetwork/GoRevive/Log"
)

type ServerManager struct {
	name          string
	socket        *gs.Socket
	eventsChannel chan gs.SocketEvent
	batchTicker   *time.Ticker
	stopTicker    chan bool
}

// New creates and starts a new ClientManager
func (sM *ServerManager) New(name string, port string) {
	var err error

	sM.socket = new(gs.Socket)
	sM.name = name
	sM.eventsChannel, err = sM.socket.New(sM.name, port, false)
	sM.stopTicker = make(chan bool, 1)

	if err != nil {
		log.Errorln(err)
	}

	go sM.run()
}

func (sM *ServerManager) run() {
	for {
		select {
		case event := <-sM.eventsChannel:
			switch {
			case event.Name == "newClient":
				go sM.newClient(event.Data.(gs.EventNewClient))
			case event.Name == "client.command":
				log.Debugf("Got event %s: %v", event.Name, event.Data.(gs.EventClientCommand).Command)
			default:
				log.Debugf("Got event %s: %v", event.Name, event.Data)
			}
		}
	}
}

func (sM *ServerManager) newClient(event gs.EventNewClient) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	log.Noteln("Client connecting")

}

func (sM *ServerManager) close(event gs.EventClientTLSClose) {
	log.Noteln("Client closed.")

	if !event.Client.State.HasLogin {
		return
	}

}

func (sM *ServerManager) error(event gs.EventClientTLSError) {
	log.Noteln("Client threw an error: ", event.Error)
}
