package fesl

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"

	"github.com/go-redis/redis"
)

// FeslManager - handles incoming and outgoing FESL data
type FeslManager struct {
	name          string
	db            *sql.DB
	redis         *redis.Client
	socket        *GameSpy.SocketTLS
	eventsChannel chan GameSpy.SocketEvent
	batchTicker   *time.Ticker
	stopTicker    chan bool
	server        bool
}

// New creates and starts a new ClientManager
func (fM *FeslManager) New(name string, port string, certFile string, keyFile string, server bool, db *sql.DB, redis *redis.Client) {
	var err error

	fM.socket = new(GameSpy.SocketTLS)
	fM.db = db
	fM.redis = redis
	fM.name = name
	fM.eventsChannel, err = fM.socket.New(fM.name, port, certFile, keyFile)
	fM.stopTicker = make(chan bool, 1)
	fM.server = server

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
				fM.newClient(event.Data.(GameSpy.EventNewClientTLS))
			case event.Name == "client.command.Hello":
				fM.hello(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.NuLogin":
				fM.NuLogin(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.NuGetPersonas":
				fM.NuGetPersonas(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.NuGetAccount":
				fM.NuGetAccount(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.NuLoginPersona":
				fM.NuLoginPersona(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.GetStatsForOwners":
				fM.GetStatsForOwners(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.GetStats":
				fM.GetStats(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.NuLookupUserInfo":
				fM.NuLookupUserInfo(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.GetPingSites":
				fM.GetPingSites(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.UpdateStats":
				fM.UpdateStats(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.GetTelemetryToken":
				fM.GetTelemetryToken(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.command.Start":
				fM.Start(event.Data.(GameSpy.EventClientTLSCommand))
			case event.Name == "client.close":
				fM.close(event.Data.(GameSpy.EventClientTLSClose))
			case event.Name == "client.command":
				fM.LogCommand(event.Data.(GameSpy.EventClientTLSCommand))
				log.Debugf("Got event %s.%s: %v", event.Name, event.Data.(GameSpy.EventClientTLSCommand).Command.Message["TXN"], event.Data.(GameSpy.EventClientTLSCommand).Command)
			default:
				log.Debugf("Got event %s: %v", event.Name, event.Data)
			}
		}
	}
}

// LogCommand - logs detailed FESL command data to a file for further analysis
func (fM *FeslManager) LogCommand(event GameSpy.EventClientTLSCommand) {
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

// MysqlRealEscapeString - you know
func MysqlRealEscapeString(value string) string {
	replace := map[string]string{"\\": "\\\\", "'": `\'`, "\\0": "\\\\0", "\n": "\\n", "\r": "\\r", `"`: `\"`, "\x1a": "\\Z"}

	for b, a := range replace {
		value = strings.Replace(value, b, a, -1)
	}

	return value
}

func (fM *FeslManager) newClient(event GameSpy.EventNewClientTLS) {
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
				memCheck := make(map[string]string)
				memCheck["TXN"] = "MemCheck"
				memCheck["memcheck.[]"] = "0"
				memCheck["salt"] = "5"
				event.Client.WriteFESL("fsys", memCheck, 0xC0000000)
				fM.logAnswer("fsys", memCheck, 0xC0000000)
			}
		}
	}()

	log.Noteln("Client connecting")

}

func (fM *FeslManager) close(event GameSpy.EventClientTLSClose) {
	log.Noteln("Client closed.")

	if event.Client.RedisState != nil {
		event.Client.RedisState.Delete()
	}

	if !event.Client.State.HasLogin {
		return
	}

}

func (fM *FeslManager) error(event GameSpy.EventClientTLSError) {
	log.Noteln("Client threw an error: ", event.Error)
}
