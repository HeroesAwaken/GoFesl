package fesl

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/HeroesAwaken/GoAwaken/core"
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
	iDB           *core.InfluxDB

	// Database Statements
	stmtGetUserByGameToken              *sql.Stmt
	stmtGetServerBySecret               *sql.Stmt
	stmtGetServerByID                   *sql.Stmt
	stmtGetServerByName                 *sql.Stmt
	stmtGetCountOfPermissionByIDAndSlug *sql.Stmt
	stmtGetHeroesByUserID               *sql.Stmt
	stmtGetHeroeByName                  *sql.Stmt
	stmtGetHeroeByID                    *sql.Stmt
	stmtClearGameServerStats            *sql.Stmt
	mapGetStatsVariableAmount           map[int]*sql.Stmt
	mapSetStatsVariableAmount           map[int]*sql.Stmt
	mapSetServerStatsVariableAmount     map[int]*sql.Stmt
}

// New creates and starts a new ClientManager
func (fM *FeslManager) New(name string, port string, certFile string, keyFile string, server bool, db *sql.DB, redis *redis.Client, iDB *core.InfluxDB) {
	var err error

	fM.socket = new(GameSpy.SocketTLS)
	fM.db = db
	fM.redis = redis
	fM.name = name
	fM.eventsChannel, err = fM.socket.New(fM.name, port, certFile, keyFile)
	fM.stopTicker = make(chan bool, 1)
	fM.server = server
	fM.iDB = iDB

	fM.mapGetStatsVariableAmount = make(map[int]*sql.Stmt)
	fM.mapSetStatsVariableAmount = make(map[int]*sql.Stmt)

	// Prepare database statements
	fM.prepareStatements()
	if err != nil {
		log.Errorln(err)
	}

	_, err = fM.stmtClearGameServerStats.Exec()
	if err != nil {
		log.Panicln("Error clearing out game server stats", err)
	}

	// Collect metrics every 10 seconds
	fM.batchTicker = time.NewTicker(time.Second * 1)
	go func() {
		for range fM.batchTicker.C {
			fM.collectMetrics()
		}
	}()

	go fM.run()
}

func (fM *FeslManager) getStatsStatement(statsAmount int) *sql.Stmt {
	var err error

	// Check if we already have a statement prepared for that amount of stats
	if statement, ok := fM.mapGetStatsVariableAmount[statsAmount]; ok {
		return statement
	}

	var query string
	for i := 1; i < statsAmount; i++ {
		query += "?, "
	}

	sql := "SELECT user_id, heroID, statsKey, statsValue" +
		"	FROM game_stats" +
		"	WHERE heroID=?" +
		"		AND user_id=?" +
		"		AND statsKey IN (" + query + "?)"

	fM.mapGetStatsVariableAmount[statsAmount], err = fM.db.Prepare(sql)
	if err != nil {
		log.Fatalln("Error preparing stmtGetStatsVariableAmount with "+sql+" query.", err.Error())
	}

	return fM.mapGetStatsVariableAmount[statsAmount]
}

func (fM *FeslManager) setStatsStatement(statsAmount int) *sql.Stmt {
	var err error

	// Check if we already have a statement prepared for that amount of stats
	if statement, ok := fM.mapSetStatsVariableAmount[statsAmount]; ok {
		return statement
	}

	var query string
	for i := 1; i < statsAmount; i++ {
		query += "(?, ?, ?, ?), "
	}

	sql := "INSERT INTO game_stats" +
		"	(user_id, heroID, statsKey, statsValue)" +
		"	VALUES " + query + "(?, ?, ?, ?)" +
		"	ON DUPLICATE KEY UPDATE" +
		"	statsValue=VALUES(statsValue)"

	fM.mapSetStatsVariableAmount[statsAmount], err = fM.db.Prepare(sql)
	if err != nil {
		log.Fatalln("Error preparing stmtSetStatsVariableAmount with "+sql+" query.", err.Error())
	}

	return fM.mapSetStatsVariableAmount[statsAmount]
}

func (fM *FeslManager) prepareStatements() {
	var err error

	fM.stmtGetUserByGameToken, err = fM.db.Prepare(
		"SELECT id, username, email, birthday, language, country, game_token" +
			"	FROM users" +
			"	WHERE game_token = ?")
	if err != nil {
		log.Fatalln("Error preparing stmtGetUserByGameToken.", err.Error())
	}

	fM.stmtGetServerBySecret, err = fM.db.Prepare(
		"SELECT game_servers.id, users.id, game_servers.servername, game_servers.secretKey, users.username" +
			"	FROM game_servers" +
			"	LEFT JOIN users" +
			"		ON users.id=game_servers.user_id" +
			"	WHERE secretKey = ?")
	if err != nil {
		log.Fatalln("Error preparing stmtGetServerBySecret.", err.Error())
	}

	fM.stmtGetServerByID, err = fM.db.Prepare(
		"SELECT game_servers.id, users.id, game_servers.servername, game_servers.secretKey, users.username" +
			"	FROM game_servers" +
			"	LEFT JOIN users" +
			"		ON users.id=game_servers.user_id" +
			"	WHERE game_servers.id = ?")
	if err != nil {
		log.Fatalln("Error preparing stmtGetServerByID.", err.Error())
	}

	fM.stmtGetServerByName, err = fM.db.Prepare(
		"SELECT game_servers.id, users.id, game_servers.servername, game_servers.secretKey, users.username" +
			"	FROM game_servers" +
			"	LEFT JOIN users" +
			"		ON users.id=game_servers.user_id" +
			"	WHERE game_servers.servername = ?")
	if err != nil {
		log.Fatalln("Error preparing stmtGetServerByName.", err.Error())
	}

	fM.stmtGetCountOfPermissionByIDAndSlug, err = fM.db.Prepare(
		"SELECT count(permissions.slug)" +
			"	FROM users" +
			"	LEFT JOIN role_user" +
			"		ON users.id=role_user.user_id" +
			"	LEFT JOIN permission_role" +
			"		ON permission_role.role_id=role_user.role_id" +
			"	LEFT JOIN permissions" +
			"		ON permissions.id=permission_role.permission_id" +
			"	WHERE users.id = ?" +
			"		AND permissions.slug = ?")
	if err != nil {
		log.Fatalln("Error preparing stmtGetCountOfPermissionByIdAndSlug.", err.Error())
	}

	fM.stmtGetHeroesByUserID, err = fM.db.Prepare(
		"SELECT id, user_id, heroName, online" +
			"	FROM game_heroes" +
			"	WHERE user_id = ?")
	if err != nil {
		log.Fatalln("Error preparing stmtGetHeroesByUserID.", err.Error())
	}

	fM.stmtGetHeroeByName, err = fM.db.Prepare(
		"SELECT id, user_id, heroName, online" +
			"	FROM game_heroes" +
			"	WHERE heroName = ?")
	if err != nil {
		log.Fatalln("Error preparing stmtGetHeroesByUserID.", err.Error())
	}

	fM.stmtGetHeroeByID, err = fM.db.Prepare(
		"SELECT id, user_id, heroName, online" +
			"	FROM game_heroes" +
			"	WHERE id = ?")
	if err != nil {
		log.Fatalln("Error preparing stmtGetHeroeByID.", err.Error())
	}

	fM.stmtClearGameServerStats, err = fM.db.Prepare(
		"DELETE FROM game_server_stats")
	if err != nil {
		log.Fatalln("Error preparing stmtClearGameServerStats.", err.Error())
	}
}

func (fM *FeslManager) closeStatements() {
	fM.stmtGetUserByGameToken.Close()
	fM.stmtGetServerBySecret.Close()
	fM.stmtGetServerByID.Close()
	fM.stmtGetServerByName.Close()
	fM.stmtGetCountOfPermissionByIDAndSlug.Close()
	fM.stmtGetHeroesByUserID.Close()
	fM.stmtGetHeroeByName.Close()
	fM.stmtClearGameServerStats.Close()

	// Close the dynamic lenght getStats statements
	for index := range fM.mapGetStatsVariableAmount {
		fM.mapGetStatsVariableAmount[index].Close()
	}

	// Close the dynamic lenght setStats statements
	for index := range fM.mapSetStatsVariableAmount {
		fM.mapSetStatsVariableAmount[index].Close()
	}
}

func (fM *FeslManager) userHasPermission(id string, slug string) bool {

	var count int
	err := fM.stmtGetCountOfPermissionByIDAndSlug.QueryRow(id, slug).Scan(&count)
	if err != nil {
		return false
	}

	// If user has at least on role allowing that permission, return true
	if count > 0 {
		return true
	}

	return false
}

func (fM *FeslManager) collectMetrics() {
	// Create a point and add to batch
	tags := map[string]string{"clients": "clients-total", "server": "feslManager" + fM.name}
	fields := map[string]interface{}{
		"clients": len(fM.socket.ClientsTLS),
	}

	fM.iDB.AddMetric("clients_total", tags, fields)
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

	// Close all database statements
	fM.closeStatements()
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
