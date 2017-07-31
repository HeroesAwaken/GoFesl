package matchmaking

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"

	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/log"
)

// Games - a list of available games
var Games = make(map[string]*GameSpy.Client)

// FindAvailableGID - returns a GID suitable for the player to join (ADD A PID HERE)
func FindAvailableGID(heroID string) string {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Get("https://heroesawaken.org/api/matchmaking/findgame/" + heroID)
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
