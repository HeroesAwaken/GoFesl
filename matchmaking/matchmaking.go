package matchmaking

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/HeroesAwaken/GoFesl/GameSpy"
	"github.com/HeroesAwaken/GoFesl/log"
)

// Games - a list of available games
var Games = make(map[string]*GameSpy.Client)

var Shard string

// FindAvailableGID - returns a GID suitable for the player to join (ADD A PID HERE)
func FindAvailableGIDs(heroID string, ip string) []string {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	url := "https://heroesawaken.com/api/mm/findgame/" + Shard + "/" + heroID + "/" + ip
	log.Noteln(url)
	resp, err := client.Get(url)
	if err != nil {
		log.Warningln("Error making request to matchmaking api")
		return make([]string, 0)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warningln("Error reading from response to matchmaking api")
		return make([]string, 0)
	}

	return strings.Split(string(body[:]), ",")
}
