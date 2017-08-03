package fesl

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/lib"
	"github.com/SpencerSharkey/GoFesl/log"
	"github.com/SpencerSharkey/GoFesl/matchmaking"
)

// Status - Basic fesl call to get overall service status (called before pnow?)
func (fM *FeslManager) Status(event GameSpy.EventClientTLSCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	// Check if user is allowed to matchmake
	if !fM.userHasPermission(event.Client.RedisState.Get("uID"), "game.matchmake") {
		log.Noteln("User not worthy: " + event.Client.RedisState.Get("username"))
		return
	}

	// Check if user has op rocket equipped
	rows, err := fM.getStatsStatement(2).Query(event.Client.RedisState.Get("heroID"), event.Client.RedisState.Get("uID"), "c_eqp", "c_apr")
	if err != nil {
		log.Errorln("Failed gettings stats for hero "+event.Client.RedisState.Get("heroID"), err.Error())
	}

	stats := make(map[string]string)
	for rows.Next() {
		var userID, heroID, statsKey, statsValue string
		err := rows.Scan(&userID, &heroID, &statsKey, &statsValue)
		if err != nil {
			log.Errorln("Issue with database:", err.Error())
		}
		stats[statsKey] = statsValue
	}
	log.Noteln(stats["c_eqp"])
	if strings.Contains(stats["c_eqp"], "3018") {
		log.Noteln("User trying to matchmake with op launcher")
		return
	}

	log.Noteln("STATUS CALLED")

	answer := make(map[string]string)
	answer["TXN"] = "Status"
	answer["id.id"] = "1"
	answer["id.partition"] = event.Command.Message["partition.partition"]
	answer["sessionState"] = "COMPLETE"
	answer["props.{}.[]"] = "2"
	answer["props.{resultType}"] = "JOIN"

	// Find latest game (do better later)
	ipint := binary.BigEndian.Uint32(event.Client.IpAddr.(*net.TCPAddr).IP.To4())
	gameIDs := matchmaking.FindAvailableGIDs(event.Client.RedisState.Get("heroID"), fmt.Sprint(ipint))

	i := 0
	for _, gid := range gameIDs {
		gameID := gid

		// Get percent_full from databse
		var args []interface{}
		statsKeys := make(map[string]string)
		args = append(args, gameID)
		args = append(args, "B-U-percent_full")

		rows, err := fM.getServerStatsVariableAmount(1).Query(args...)
		if err != nil {
			log.Errorln("Failed gettings stats for hero "+gameID, err.Error())
		}

		for rows.Next() {
			var gid, statsKey, statsValue string
			err := rows.Scan(&gid, &statsKey, &statsValue)
			if err != nil {
				log.Errorln("Issue with database:", err.Error())
			}
			statsKeys[statsKey] = statsValue
		}

		if statsKeys["B-U-percent_full"] == "100" {
			// Don't add for matchmaking if server is full
			continue
		}

		gameServer := new(lib.RedisObject)
		gameServer.New(fM.redis, "gdata", gameID)

		answer["props.{games}."+strconv.Itoa(i)+".lid"] = "1"
		//answer["props.{games}."+strconv.Itoa(i)+".fit"] = strconv.Itoa(len(matchmaking.Games) - i)
		answer["props.{games}."+strconv.Itoa(i)+".fit"] = "1000"
		answer["props.{games}."+strconv.Itoa(i)+".gid"] = gameID

		log.Noteln(gameServer.Get("NAME") + " GID: " + gameID + " with fitness of: " + strconv.Itoa(len(matchmaking.Games)-i))
		i++
	}

	answer["props.{games}.[]"] = strconv.Itoa(i)

	/*
		answer["props.{games}.0.lid"] = "1"
		answer["props.{games}.0.fit"] = "1001"
		answer["props.{games}.0.gid"] = gameID
		answer["props.{games}.[]"] = "1"
	*/
	/*
		answer["props.{games}.1.lid"] = "2"
		answer["props.{games}.1.fit"] = "100"
		answer["props.{games}.1.gid"] = "2"
		answer["props.{games}.1.avgFit"] = "100"
	*/

	event.Client.WriteFESL("pnow", answer, 0x80000000)
	fM.logAnswer("pnow", answer, 0x80000000)
}
