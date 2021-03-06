package fesl

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/HeroesAwaken/GoFesl/GameSpy"
	"github.com/HeroesAwaken/GoFesl/log"
	"github.com/HeroesAwaken/GoFesl/matchmaking"
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
		fM.sendDenied(event)
		return
	}

	// Check if user has op rocket equipped
	rows, err := fM.getStatsStatement(2).Query(event.Client.RedisState.Get("heroID"), event.Client.RedisState.Get("uID"), "c_eqp", "c_apr")
	if err != nil {
		log.Errorln("Failed gettings stats for hero "+event.Client.RedisState.Get("heroID"), err.Error())
		fM.sendDenied(event)
		return
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

	if strings.Contains(stats["c_eqp"], "3018") {
		fM.sendDenied(event)
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

	for i, gid := range gameIDs {
		answer["props.{games}."+strconv.Itoa(i)+".lid"] = "1"
		answer["props.{games}."+strconv.Itoa(i)+".fit"] = "1000"
		answer["props.{games}."+strconv.Itoa(i)+".gid"] = gid
	}

	answer["props.{games}.[]"] = strconv.Itoa(len(gameIDs))

	event.Client.WriteFESL("pnow", answer, 0x80000000)
	fM.logAnswer("pnow", answer, 0x80000000)
}

func (fM *FeslManager) sendDenied(event GameSpy.EventClientTLSCommand) {
	answer := make(map[string]string)
	answer["TXN"] = "Status"
	answer["id.id"] = "1"
	answer["id.partition"] = event.Command.Message["partition.partition"]
	answer["sessionState"] = "COMPLETE"
	answer["props.{}.[]"] = "2"
	answer["props.{resultType}"] = "JOIN"
	answer["props.{games}.[]"] = "0"
	event.Client.WriteFESL("pnow", answer, 0x80000000)
	fM.logAnswer("pnow", answer, 0x80000000)
}
