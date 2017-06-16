package main

import (
	"time"

	gs "github.com/ReviveNetwork/GoRevive/GameSpy"
	log "github.com/ReviveNetwork/GoRevive/Log"
)

type FeslManager struct {
	name          string
	socket        *gs.SocketTLS
	eventsChannel chan gs.SocketEvent
	batchTicker   *time.Ticker
	stopTicker    chan bool
}

// New creates and starts a new ClientManager
func (fM *FeslManager) New(name string, port string, certFile string, keyFile string) {
	var err error

	fM.socket = new(gs.SocketTLS)
	fM.name = name
	fM.eventsChannel, err = fM.socket.New(fM.name, port, certFile, keyFile)
	fM.stopTicker = make(chan bool, 1)

	if err != nil {
		log.Errorln(err)
	}

	go fM.run()
}

func (fM *FeslManager) run() {
	for {
		select {
		case event := <-fM.eventsChannel:
			switch {
			case event.Name == "newClient":
				go fM.newClient(event.Data.(gs.EventNewClientTLS))
			case event.Name == "client.command.Hello":
				go fM.hello(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuLogin":
				go fM.NuLogin(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuGetPersonas":
				go fM.NuGetPersonas(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuGetAccount":
				go fM.NuGetAccount(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuLoginPersona":
				go fM.NuLoginPersona(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.GetStats":
				go fM.GetStats(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuLookupUserInfo":
				go fM.NuLookupUserInfo(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.GetPingSites":
				go fM.GetPingSites(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.UpdateStats":
				go fM.UpdateStats(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.close":
				go fM.close(event.Data.(gs.EventClientTLSClose))
			case event.Name == "client.command":
				log.Debugf("Got event %s.%s: %v", event.Name, event.Data.(gs.EventClientTLSCommand).Command.Message["TXN"], event.Data.(gs.EventClientTLSCommand).Command)
			default:
				log.Debugf("Got event %s: %v", event.Name, event.Data)
			}
		}
	}
}

func (fM *FeslManager) UpdateStats(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TXN"] = "UpdateStats"
	event.Client.WriteFESL(event.Command.Query, answerPacket, event.Command.PayloadID)
}

func (fM *FeslManager) GetPingSites(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TXN"] = "GetPingSites"
	answerPacket["minPingSitesToPing"] = "0"
	answerPacket["pingSites.[]"] = "3"
	answerPacket["pingSites.0.addr"] = "5.9.107.138"
	answerPacket["pingSites.0.name"] = "eu-ip"
	answerPacket["pingSites.0.type"] = "0"
	answerPacket["pingSites.1.addr"] = "5.9.107.138"
	answerPacket["pingSites.1.name"] = "ec-ip"
	answerPacket["pingSites.1.type"] = "0"
	answerPacket["pingSites.2.addr"] = "5.9.107.138"
	answerPacket["pingSites.2.name"] = "wc-ip"
	answerPacket["pingSites.2.type"] = "0"
	event.Client.WriteFESL(event.Command.Query, answerPacket, event.Command.PayloadID)
}

func (fM *FeslManager) NuLoginPersona(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuLoginPersona"
	loginPacket["lkey"] = "12345"
	loginPacket["profileId"] = "0"
	loginPacket["userId"] = "0"
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
}

func (fM *FeslManager) NuLogin(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuLogin"
	loginPacket["displayName"] = "MakaHost"
	loginPacket["entitledGameFeatureWrappers.status"] = "0"
	loginPacket["entitledGameFeatureWrappers.message"] = ""
	loginPacket["entitledGameFeatureWrappers.entitlementExpirationDate"] = ""
	loginPacket["entitlementExpirationDays"] = "-1"
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
}

func (fM *FeslManager) NuLookupUserInfo(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	personaPacket := make(map[string]string)
	personaPacket["TXN"] = "NuLookupUserInfo"
	personaPacket["user"] = "1"
	personaPacket["userInfo.[]"] = "2"
	personaPacket["userInfo.0.userName"] = "default"
	personaPacket["userInfo.0.userId"] = "0"
	personaPacket["userInfo.0.xuid"] = "0"
	personaPacket["userInfo.0.masterUserId"] = "0"
	personaPacket["userInfo.0.namespace"] = "MAIN"
	personaPacket["userInfo.1.userName"] = "MakaHost"
	personaPacket["userInfo.1.userId"] = "1"
	personaPacket["userInfo.1.xuid"] = "1"
	personaPacket["userInfo.1.masterUserId"] = "0"
	personaPacket["userInfo.1.namespace"] = "MAIN"
	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
}

func (fM *FeslManager) NuGetPersonas(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	personaPacket := make(map[string]string)
	personaPacket["TXN"] = "NuGetPersonas"
	personaPacket["personas.0"] = "default"
	personaPacket["personas.1"] = "MakaHost"
	personaPacket["personas.[]"] = "2"
	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
}

func (fM *FeslManager) NuGetAccount(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuGetAccount"
	loginPacket["heroName"] = "MakaHost"
	loginPacket["nuid"] = "max@bosse.io"
	loginPacket["DOBDay"] = "1"
	loginPacket["DOBMonthg"] = "1"
	loginPacket["DOBYear"] = "2017"
	loginPacket["userId"] = "1"
	loginPacket["globalOptin"] = "0"
	loginPacket["thidPartyOptin"] = "0"
	loginPacket["language"] = "en"
	loginPacket["country"] = "DE"
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
}

func (fM *FeslManager) GetStats(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "GetStats"
	loginPacket["stats.[]"] = "2"
	loginPacket["stats.0"] = "MakaHost"
	loginPacket["stats.0.ownerId"] = "0"
	loginPacket["stats.0.ownerType"] = "0"
	loginPacket["stats.0.stats.[]"] = "1"
	loginPacket["stats.0.stats.0"] = "5"
	loginPacket["stats.0.stats.1"] = "6"
	loginPacket["stats.0.stats.2"] = "7"
	loginPacket["stats.0.stats.3"] = "8"
	loginPacket["stats.0.stats.4"] = "9"
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
}

func (fM *FeslManager) hello(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	getSession := make(map[string]string)
	getSession["TXN"] = "GetSessionId"
	event.Client.WriteFESL("gsum", getSession, 0)

	helloPacket := make(map[string]string)
	helloPacket["TXN"] = "Hello"
	helloPacket["domainPartition.domain"] = "eagames"
	helloPacket["domainPartition.subDomain"] = "bfwest-dedicated"
	helloPacket["curTime"] = "Jun-15-2017 07:26:12 UTC"
	helloPacket["activityTimeoutSecs"] = "0"
	helloPacket["messengerIp"] = "messaging.ea.com"
	helloPacket["messengerPort"] = "13505"
	helloPacket["theaterIp"] = "bfwest-dedicated.theater.ea.com"
	helloPacket["theaterPort"] = "18275"
	event.Client.WriteFESL("fsys", helloPacket, 0xC0000001)

}

func (fM *FeslManager) newClient(event gs.EventNewClientTLS) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	memCheck := make(map[string]string)
	memCheck["TXN"] = "MemCheck"
	memCheck["memcheck.[]"] = "0"
	memCheck["salt"] = "5"
	event.Client.WriteFESL("fsys", memCheck, 0xC0000000)

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
				pingSite := make(map[string]string)
				pingSite["TXN"] = "Ping"
				event.Client.WriteFESL("fsys", pingSite, 0)
			}
		}
	}()

	log.Noteln("Client connecting")

}

func (fM *FeslManager) close(event gs.EventClientTLSClose) {
	log.Noteln("Client closed.")

	if !event.Client.State.HasLogin {
		return
	}

}

func (fM *FeslManager) error(event gs.EventClientTLSError) {
	log.Noteln("Client threw an error: ", event.Error)
}
