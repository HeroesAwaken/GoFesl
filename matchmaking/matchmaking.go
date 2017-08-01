package matchmaking

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// Games - a list of available games
var Games = make(map[string]*GameSpy.Client)

var Shard string

// FindAvailableGID - returns a GID suitable for the player to join (ADD A PID HERE)
func FindAvailableGID(heroID string, ip net.IP) string {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	var ipint uint32
	if len(ip) == 16 {
		ipint = binary.BigEndian.Uint32(ip[12:16])
	}
	ipint = binary.BigEndian.Uint32(ip)

	client := &http.Client{Transport: tr}
	resp, err := client.Get("https://heroesawaken.org/api/mm/findgame/" + Shard + "/" + heroID + "/" + fmt.Sprint(ipint))
	if err != nil {
		log.Warningln("Error making request to matchmaking api")
		return "0"
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warningln("Error reading from response to matchmaking api")
		return "0"
	}

	return string(body[:])
}
