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
				fM.newClient(event.Data.(gs.EventNewClientTLS))
			case event.Name == "client.command.Hello":
				fM.hello(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuLogin":
				fM.NuLogin(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuGetPersonas":
				fM.NuGetPersonas(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuGetAccount":
				fM.NuGetAccount(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuLoginPersona":
				fM.NuLoginPersona(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.GetStatsForOwners":
				fM.GetStatsForOwners(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.GetStats":
				fM.GetStats(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.NuLookupUserInfo":
				fM.NuLookupUserInfo(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.GetPingSites":
				fM.GetPingSites(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.UpdateStats":
				fM.UpdateStats(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.GetTelemetryToken":
				fM.GetTelemetryToken(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.command.Start":
				fM.Start(event.Data.(gs.EventClientTLSCommand))
			case event.Name == "client.close":
				fM.close(event.Data.(gs.EventClientTLSClose))
			case event.Name == "client.command":
				fM.LogCommand(event.Data.(gs.EventClientTLSCommand))
				log.Debugf("Got event %s.%s: %v", event.Name, event.Data.(gs.EventClientTLSCommand).Command.Message["TXN"], event.Data.(gs.EventClientTLSCommand).Command)
			default:
				log.Debugf("Got event %s: %v", event.Name, event.Data)
			}
		}
	}
}

func (fM *FeslManager) LogCommand(event gs.EventClientTLSCommand) {
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

func (fM *FeslManager) logAnswer(msgType string, msgContent map[string]string, msgType2 uint32) {
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

func (fM *FeslManager) GetTelemetryToken(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TXN"] = "GetTelemetryToken"
	answerPacket["telemetryToken"] = "MTU5LjE1My4yMzUuMjYsOTk0NixlblVTLF7ZmajcnLfGpKSJk53K/4WQj7LRw9asjLHvxLGhgoaMsrDE3bGWhsyb4e6woYKGjJiw4MCBg4bMsrnKibuDppiWxYKditSp0amvhJmStMiMlrHk4IGzhoyYsO7A4dLM26rTgAo%3d"
	answerPacket["enabled"] = "US"
	answerPacket["filters"] = ""
	answerPacket["disabled"] = ""
	event.Client.WriteFESL(event.Command.Query, answerPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answerPacket, event.Command.PayloadID)
}

func (fM *FeslManager) Status(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TXN"] = "Status"
	answerPacket["id.id"] = "605"
	answerPacket["id.partition"] = event.Command.Message["partition.partition"]
	answerPacket["sessionState"] = "COMPLETE"
	answerPacket["props.{}"] = "3"
	answerPacket["props.{resultType}"] = "LIST"
	answerPacket["props.{availableServerCount}"] = "0"
	answerPacket["props.{avgFit}"] = "100"

	answerPacket["props.{games}.[]"] = "1"
	answerPacket["props.{games}.0.lid"] = "0"
	answerPacket["props.{games}.0.gid"] = "0"
	answerPacket["props.{games}.0.fit"] = "0"
	answerPacket["props.{games}.0.avgFit"] = "0"
	/*
		answerPacket["props.{games}.1.lid"] = "2"
		answerPacket["props.{games}.1.fit"] = "100"
		answerPacket["props.{games}.1.gid"] = "2"
		answerPacket["props.{games}.1.avgFit"] = "100"
	*/

	event.Client.WriteFESL("pnow", answerPacket, 0x80000000)
	fM.logAnswer("pnow", answerPacket, 0x80000000)
}

func (fM *FeslManager) Start(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TXN"] = "Start"
	answerPacket["id.id"] = "605"
	answerPacket["id.partition"] = event.Command.Message["partition.partition"]
	event.Client.WriteFESL(event.Command.Query, answerPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answerPacket, event.Command.PayloadID)

	fM.Status(event)
}

func (fM *FeslManager) UpdateStats(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := event.Command.Message
	answerPacket["TXN"] = "UpdateStats"

	answerPacket["u.0.s.1.k"] = "c_wallet_hero"
	answerPacket["u.0.s.1.v"] = "5"
	event.Client.WriteFESL(event.Command.Query, answerPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answerPacket, event.Command.PayloadID)
}

func (fM *FeslManager) GetPingSites(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	answerPacket := make(map[string]string)
	answerPacket["TXN"] = "GetPingSites"
	answerPacket["minPingSitesToPing"] = "0"
	answerPacket["pingSites.[]"] = "1"
	answerPacket["pingSites.0.addr"] = "127.0.0.1"
	answerPacket["pingSites.0.name"] = "eu-ip"
	answerPacket["pingSites.0.type"] = "0"
	/*
		answerPacket["pingSites.1.addr"] = "5.9.107.138"
		answerPacket["pingSites.1.name"] = "ec-ip"
		answerPacket["pingSites.1.type"] = "0"
		answerPacket["pingSites.2.addr"] = "5.9.107.138"
		answerPacket["pingSites.2.name"] = "wc-ip"
		answerPacket["pingSites.2.type"] = "0"
	*/
	event.Client.WriteFESL(event.Command.Query, answerPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, answerPacket, event.Command.PayloadID)
}

func (fM *FeslManager) NuLoginPersona(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "NuLoginPersona"
	loginPacket["lkey"] = "12345"
	loginPacket["profileId"] = "1"
	loginPacket["userId"] = "1"
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
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
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
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
	personaPacket["userInfo.0.userName"] = "MakaHost"
	personaPacket["userInfo.0.userId"] = "1"
	personaPacket["userInfo.0.xuid"] = "1"
	personaPacket["userInfo.0.masterUserId"] = "1"
	personaPacket["userInfo.0.namespace"] = "MAIN"
	personaPacket["userInfo.1.userName"] = "MakaHost2"
	personaPacket["userInfo.1.userId"] = "2"
	personaPacket["userInfo.1.xuid"] = "1"
	personaPacket["userInfo.1.masterUserId"] = "1"
	personaPacket["userInfo.1.namespace"] = "MAIN"
	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)
}

func (fM *FeslManager) NuGetPersonas(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	personaPacket := make(map[string]string)
	personaPacket["TXN"] = "NuGetPersonas"
	personaPacket["personas.0"] = "MakaHost"
	personaPacket["personas.1"] = "MakaHost2"
	personaPacket["personas.[]"] = "2"
	event.Client.WriteFESL(event.Command.Query, personaPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, personaPacket, event.Command.PayloadID)
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
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}

func (fM *FeslManager) GetStatsForOwners(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "GetStats"

	loginPacket["stats.[]"] = "2"
	loginPacket["stats.0.ownerId"] = "1"
	loginPacket["stats.0.ownerType"] = "1"
	loginPacket["stats.0.stats.[]"] = "11"
	loginPacket["stats.0.stats.0.key"] = "elo"
	loginPacket["stats.0.stats.0.value"] = "1"
	loginPacket["stats.0.stats.1.key"] = "xp"
	loginPacket["stats.0.stats.1.value"] = "500"
	loginPacket["stats.0.stats.2.key"] = "level"
	loginPacket["stats.0.stats.2.value"] = "2"
	loginPacket["stats.0.stats.3.key"] = "c_apr"
	loginPacket["stats.0.stats.3.value"] = "3"
	loginPacket["stats.0.stats.4.key"] = "c_fhrs"
	loginPacket["stats.0.stats.4.value"] = "0"
	loginPacket["stats.0.stats.5.key"] = "c_hrs"
	loginPacket["stats.0.stats.5.value"] = "0"
	loginPacket["stats.0.stats.6.key"] = "c_hrc"
	loginPacket["stats.0.stats.6.value"] = "0"
	loginPacket["stats.0.stats.7.key"] = "c_skc"
	loginPacket["stats.0.stats.7.value"] = "0"
	loginPacket["stats.0.stats.8.key"] = "c_ft"
	loginPacket["stats.0.stats.8.value"] = "0"
	loginPacket["stats.0.stats.9.key"] = "c_kit"
	loginPacket["stats.0.stats.9.value"] = "0"
	loginPacket["stats.0.stats.10.key"] = "c_team"
	loginPacket["stats.0.stats.10.value"] = "1"

	loginPacket["stats.1.ownerId"] = "2"
	loginPacket["stats.1.ownerType"] = "1"
	loginPacket["stats.1.stats.[]"] = "11"
	loginPacket["stats.1.stats.0.key"] = "elo"
	loginPacket["stats.1.stats.0.value"] = "2"
	loginPacket["stats.1.stats.1.key"] = "xp"
	loginPacket["stats.1.stats.1.value"] = "5000"
	loginPacket["stats.1.stats.2.key"] = "level"
	loginPacket["stats.1.stats.2.value"] = "1"
	loginPacket["stats.1.stats.3.key"] = "c_apr"
	loginPacket["stats.1.stats.3.value"] = "1"
	loginPacket["stats.1.stats.4.key"] = "c_fhrs"
	loginPacket["stats.1.stats.4.value"] = "1"
	loginPacket["stats.1.stats.5.key"] = "c_hrs"
	loginPacket["stats.1.stats.5.value"] = "1"
	loginPacket["stats.1.stats.6.key"] = "c_hrc"
	loginPacket["stats.1.stats.6.value"] = "1"
	loginPacket["stats.1.stats.7.key"] = "c_skc"
	loginPacket["stats.1.stats.7.value"] = "1"
	loginPacket["stats.1.stats.8.key"] = "c_ft"
	loginPacket["stats.1.stats.8.value"] = "1"
	loginPacket["stats.1.stats.9.key"] = "c_kit"
	loginPacket["stats.1.stats.9.value"] = "1"
	loginPacket["stats.1.stats.10.key"] = "c_team"
	loginPacket["stats.1.stats.10.value"] = "2"

	event.Client.WriteFESL(event.Command.Query, loginPacket, 0xC0000007)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)
}

func (fM *FeslManager) GetStats(event gs.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	loginPacket := make(map[string]string)
	loginPacket["TXN"] = "GetStats"
	loginPacket["ownerId"] = event.Command.Message["owner"]
	loginPacket["ownerType"] = "1"
	loginPacket["periodPast"] = "0"
	loginPacket["periodId"] = "0"

	loginPacket["stats.[]"] = event.Command.Message["keys.[]"]
	keys, _ := strconv.Atoi(event.Command.Message["keys.[]"])
	for i := 0; i < keys; i++ {
		loginPacket["stats."+strconv.Itoa(i)+".key"] = event.Command.Message["keys."+strconv.Itoa(i)+""]
		loginPacket["stats."+strconv.Itoa(i)+".value"] = "99.0"
	}
	/*
		loginPacket["stats.[]"] = event.Command.Message["stats.[]"]
		loginPacket["stats.0.key"] = "c_ltm"
		loginPacket["stats.0.value"] = "1"
		loginPacket["stats.1.key"] = "c_slm"
		loginPacket["stats.1.value"] = "1"
		loginPacket["stats.1.key"] = "c_tut"
		loginPacket["stats.1.value"] = "1"
	*/
	event.Client.WriteFESL(event.Command.Query, loginPacket, event.Command.PayloadID)
	fM.logAnswer(event.Command.Query, loginPacket, event.Command.PayloadID)

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
	fM.logAnswer("fsys", helloPacket, 0xC0000001)

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
	fM.logAnswer("fsys", memCheck, 0xC0000000)

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
