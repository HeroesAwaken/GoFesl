package theater

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/HeroesAwaken/GoAwaken/core"
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/lib"
	"github.com/SpencerSharkey/GoFesl/log"
	"github.com/go-redis/redis"
)

// GameClient Represents a game client connected to theater
type GameClient struct {
	ip   string
	port string
}

// GameServer Represents a game server and it's data
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

// TheaterManager Handles incoming and outgoing theater communication
type TheaterManager struct {
	name             string
	socket           *GameSpy.Socket
	socketUDP        *GameSpy.SocketUDP
	db               *sql.DB
	redis            *redis.Client
	eventsChannel    chan GameSpy.SocketEvent
	eventsChannelUDP chan GameSpy.SocketUDPEvent
	batchTicker      *time.Ticker
	stopTicker       chan bool
	cacheCounters    *lib.RedisObject
	iDB              *core.InfluxDB
	localMode        bool

	// Database Statements
	stmtGetHeroeByID                      *sql.Stmt
	stmtDelteServerStatsByGID             *sql.Stmt
	mapGetStatsVariableAmount             map[int]*sql.Stmt
	mapSetServerStatsVariableAmount       map[int]*sql.Stmt
	mapSetServerPlayerStatsVariableAmount map[int]*sql.Stmt
}

const COUNTER_GID_KEY = "counters:GID"

// New creates and starts a new TheaterManager
func (tM *TheaterManager) New(name string, port string, db *sql.DB, redis *redis.Client, iDB *core.InfluxDB, localMode bool) {
	var err error

	tM.socket = new(GameSpy.Socket)
	tM.socketUDP = new(GameSpy.SocketUDP)
	tM.db = db
	tM.redis = redis
	tM.name = name
	tM.eventsChannel, err = tM.socket.New(tM.name, port, true)
	tM.iDB = iDB
	tM.localMode = localMode
	if err != nil {
		log.Errorln(err)
	}
	tM.eventsChannelUDP, err = tM.socketUDP.New(tM.name, port, true)
	if err != nil {
		log.Errorln(err)
	}
	tM.stopTicker = make(chan bool, 1)

	// Prepare database statements
	tM.mapGetStatsVariableAmount = make(map[int]*sql.Stmt)
	tM.mapSetServerStatsVariableAmount = make(map[int]*sql.Stmt)
	tM.mapSetServerPlayerStatsVariableAmount = make(map[int]*sql.Stmt)
	tM.prepareStatements()

	// Collect metrics every 10 seconds
	tM.batchTicker = time.NewTicker(time.Second * 1)
	go func() {
		for range tM.batchTicker.C {
			tM.collectMetrics()
		}
	}()

	//tM.redis.Set(COUNTER_GID_KEY, 0, 0)

	go tM.run()
}

func (tM *TheaterManager) prepareStatements() {
	var err error

	tM.stmtGetHeroeByID, err = tM.db.Prepare(
		"SELECT id, user_id, heroName, online" +
			"	FROM game_heroes" +
			"	WHERE id = ?")
	if err != nil {
		log.Fatalln("Error preparing stmtGetHeroeByID.", err.Error())
	}

	tM.stmtDelteServerStatsByGID, err = tM.db.Prepare(
		"DELETE FROM game_server_stats WHERE gid = ß")
	if err != nil {
		log.Fatalln("Error preparing stmtClearGameServerStats.", err.Error())
	}
}

func (tM *TheaterManager) getStatsStatement(statsAmount int) *sql.Stmt {
	var err error

	// Check if we already have a statement prepared for that amount of stats
	if statement, ok := tM.mapGetStatsVariableAmount[statsAmount]; ok {
		return statement
	}

	var query string
	for i := 1; i < statsAmount; i++ {
		query += "?, "
	}

	sql := "SELECT game_heroes.user_id, game_heroes.id, game_heroes.heroName, game_stats.statsKey, game_stats.statsValue" +
		"	FROM game_heroes" +
		"	LEFT JOIN game_stats" +
		"		ON game_stats.user_id = game_heroes.user_id" +
		"		AND game_stats.heroID = game_heroes.id" +
		"	WHERE game_heroes.id=?" +
		"		AND game_stats.statsKey IN (" + query + "?)"

	tM.mapGetStatsVariableAmount[statsAmount], err = tM.db.Prepare(sql)
	if err != nil {
		log.Fatalln("Error preparing stmtGetStatsVariableAmount with "+sql+" query.", err.Error())
	}

	return tM.mapGetStatsVariableAmount[statsAmount]
}

func (tM *TheaterManager) setServerStatsStatement(statsAmount int) *sql.Stmt {
	var err error

	// Check if we already have a statement prepared for that amount of stats
	if statement, ok := tM.mapSetServerStatsVariableAmount[statsAmount]; ok {
		return statement
	}

	var query string
	for i := 1; i < statsAmount; i++ {
		query += "(?, ?, ?, NOW()), "
	}

	sql := "INSERT INTO game_server_stats" +
		"	(gid, statsKey, statsValue, created_at)" +
		"	VALUES " + query + "(?, ?, ?, NOW())" +
		"	ON DUPLICATE KEY UPDATE" +
		"	statsValue=VALUES(statsValue)," +
		"   updated_at=NOW()"

	tM.mapSetServerStatsVariableAmount[statsAmount], err = tM.db.Prepare(sql)
	if err != nil {
		log.Fatalln("Error preparing setServerStatsStatement with "+sql+" query.", err.Error())
	}

	return tM.mapSetServerStatsVariableAmount[statsAmount]
}

func (tM *TheaterManager) setServerPlayerStatsStatement(statsAmount int) *sql.Stmt {
	var err error

	// Check if we already have a statement prepared for that amount of stats
	if statement, ok := tM.mapSetServerPlayerStatsVariableAmount[statsAmount]; ok {
		return statement
	}

	var query string
	for i := 1; i < statsAmount; i++ {
		query += "(?, ?, ?, ?, NOW()), "
	}

	sql := "INSERT INTO game_server_player_stats" +
		"	(gid, pid, statsKey, statsValue, created_at)" +
		"	VALUES " + query + "(?, ?, ?, ?, NOW())" +
		"	ON DUPLICATE KEY UPDATE" +
		"	statsValue=VALUES(statsValue)," +
		"   updated_at=NOW()"

	tM.mapSetServerPlayerStatsVariableAmount[statsAmount], err = tM.db.Prepare(sql)
	if err != nil {
		log.Fatalln("Error preparing mapSetServerPlayerStatsVariableAmount with "+sql+" query.", err.Error())
	}

	return tM.mapSetServerPlayerStatsVariableAmount[statsAmount]
}

func (tM *TheaterManager) closeStatements() {
	// Close the dynamic lenght getStats statements
	for index := range tM.mapGetStatsVariableAmount {
		tM.mapGetStatsVariableAmount[index].Close()
	}
}

func (tM *TheaterManager) collectMetrics() {
	// Create a point and add to batch
	tags := map[string]string{"clients": "clients-total", "server": "theaterManager-" + tM.name}
	fields := map[string]interface{}{
		"clients": len(tM.socket.Clients),
	}

	tM.iDB.AddMetric("clients_total", tags, fields)
}

func (tM *TheaterManager) run() {
	for {
		select {
		case event := <-tM.eventsChannelUDP:
			switch {
			case event.Name == "command.ECHO":
				go tM.ECHO(event)
			case event.Name == "command":
				tM.LogCommandUDP(event.Data.(*GameSpy.CommandFESL))
				log.Debugf("UDP Got event %s: %v", event.Name, event.Data.(*GameSpy.CommandFESL))
			default:
				log.Debugf("UDP Got event %s: %v", event.Name, event.Data)
			}
		case event := <-tM.eventsChannel:
			switch {
			case event.Name == "newClient":
				go tM.newClient(event.Data.(GameSpy.EventNewClient))
			case event.Name == "client.command.CONN":
				go tM.CONN(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.USER":
				go tM.USER(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.LLST":
				go tM.LLST(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.GDAT":
				go tM.GDAT(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.EGAM":
				go tM.EGAM(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.ECNL":
				go tM.ECNL(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.CGAM":
				go tM.CGAM(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.UBRA":
				go tM.UBRA(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.UGAM":
				go tM.UGAM(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.EGRS":
				go tM.EGRS(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.GLST":
				go tM.GLST(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.PENT":
				go tM.PENT(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command.UPLA":
				go tM.UPLA(event.Data.(GameSpy.EventClientFESLCommand))
			case event.Name == "client.command":
				tM.LogCommand(event.Data.(GameSpy.EventClientFESLCommand))
				log.Debugf("Got event %s: %v", event.Name, event.Data.(GameSpy.EventClientFESLCommand).Command)
			default:
				log.Debugf("Got event %s: %v", event.Name, event.Data)
			}
		}
	}

	// Close all database statements
	tM.closeStatements()
}

// LogCommandUDP log data to a debug file for further analysis
func (tM *TheaterManager) LogCommandUDP(event *GameSpy.CommandFESL) {
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

// LogCommand log data to a debug file for further analysis
func (tM *TheaterManager) LogCommand(event GameSpy.EventClientFESLCommand) {
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

func (tM *TheaterManager) newClient(event GameSpy.EventNewClient) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}
	log.Noteln("Client connecting")

	// Start Heartbeat
	event.Client.State.HeartTicker = time.NewTicker(time.Second * 15)
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

func (tM *TheaterManager) close(event GameSpy.EventClientTLSClose) {
	log.Noteln("Client closed.")

	if event.Client.RedisState.Get("gdata:GID") != "" {
		// Delete game from db

		_, err := tM.stmtDelteServerStatsByGID.Exec(event.Client.RedisState.Get("gdata:GID"))
		if err != nil {
			log.Errorln("Failed deleting settings for  "+event.Client.RedisState.Get("gdata:GID"), err.Error())
		}

		gameServer := new(lib.RedisObject)
		gameServer.New(tM.redis, "gdata", event.Client.RedisState.Get("gdata:GID"))
		gameServer.Delete()
	}

	if event.Client.RedisState != nil {
		event.Client.RedisState.Delete()
	}

	if !event.Client.State.HasLogin {
		return
	}

}

func (tM *TheaterManager) error(event GameSpy.EventClientTLSError) {
	log.Noteln("Client threw an error: ", event.Error)
}
