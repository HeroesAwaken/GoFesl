package theater

import (
	"net"
	"strconv"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/lib"
	"github.com/SpencerSharkey/GoFesl/log"
	"github.com/SpencerSharkey/GoFesl/matchmaking"
)

// CGAM - SERVER called to create a game
func (tM *TheaterManager) CGAM(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	addr, ok := event.Client.IpAddr.(*net.TCPAddr)

	if !ok {
		log.Errorln("Failed turning IpAddr to net.TCPAddr")
		return
	}

	gameIDInt, _ := tM.redis.Incr(COUNTER_GID_KEY).Result()
	gameID := strconv.Itoa(int(gameIDInt))

	// Store our server for easy access later
	matchmaking.Games[gameID] = event.Client

	var args []interface{}

	// Setup a new key for our game
	gameServer := new(lib.RedisObject)
	gameServer.New(tM.redis, "gdata", gameID)

	keys := 0

	// Stores what we know about this game in the redis db
	for index, value := range event.Command.Message {
		if index == "TID" {
			continue
		}

		keys++

		// Strip quotes
		if len(value) > 0 && value[0] == '"' {
			value = value[1:]
		}
		if len(value) > 0 && value[len(value)-1] == '"' {
			value = value[:len(value)-1]
		}
		gameServer.Set(index, value)

		args = append(args, gameID)
		args = append(args, index)
		args = append(args, value)
	}

	gameServer.Set("LID", "1")
	gameServer.Set("GID", gameID)
	gameServer.Set("IP", addr.IP.String())
	gameServer.Set("ACTIVE-PLAYERS", "0")
	gameServer.Set("QUEUE-LENGTH", "0")

	event.Client.RedisState.Set("gdata:GID", gameID)

	var err error
	_, err = tM.setServerStatsStatement(keys).Exec(args...)
	if err != nil {
		log.Errorln("Failed setting stats for game server "+gameID, err.Error())
	}

	answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	answer["LID"] = "1"
	answer["UGID"] = event.Command.Message["UGID"]
	answer["MAX-PLAYERS"] = event.Command.Message["MAX-PLAYERS"] // Validate this
	answer["EKEY"] = "O65zZ2D2A58mNrZw1hmuJw%3d%3d"              // Eventually generate this
	answer["UGID"] = event.Command.Message["UGID"]               // Verify these against some auth shit
	answer["SECRET"] = "2587913"                                 // Eventually generate this too
	answer["JOIN"] = event.Command.Message["JOIN"]
	answer["J"] = event.Command.Message["JOIN"]
	answer["GID"] = gameID
	event.Client.WriteFESL("CGAM", answer, 0x0)
	tM.logAnswer("CGAM", answer, 0x0)

	// Create game in database
	_, err = tM.stmtAddGame.Exec(gameID, Shard, addr.IP.String(), event.Command.Message["PORT"], event.Command.Message["B-version"], event.Command.Message["JOIN"], event.Command.Message["B-U-map"], 0, 0, event.Command.Message["MAX-PLAYERS"], 0, 0, "")
	if err != nil {
		log.Panicln(err)
	}
}
