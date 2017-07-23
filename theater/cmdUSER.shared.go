package theater

import (
	"github.com/HeroesAwaken/GoAwaken/core"
	"github.com/SpencerSharkey/GoFesl/GameSpy"
	"github.com/SpencerSharkey/GoFesl/lib"
	"github.com/SpencerSharkey/GoFesl/log"
)

// USER - SHARED Called to get user data about client? No idea
func (tM *TheaterManager) USER(event GameSpy.EventClientFESLCommand) {
	if !event.Client.IsActive {
		log.Noteln("Client left")
		return
	}

	lkeyRedis := new(lib.RedisObject)
	lkeyRedis.New(tM.redis, "lkeys", event.Command.Message["LKEY"])

	redisState := new(core.RedisState)
	redisState.New(tM.redis, "mm:"+event.Command.Message["LKEY"])
	event.Client.RedisState = redisState

	redisState.Set("id", lkeyRedis.Get("id"))
	redisState.Set("userID", lkeyRedis.Get("userID"))
	redisState.Set("name", lkeyRedis.Get("name"))

	answer := make(map[string]string)
	answer["TID"] = event.Command.Message["TID"]
	answer["NAME"] = lkeyRedis.Get("name")
	answer["CID"] = ""
	event.Client.WriteFESL(event.Command.Query, answer, 0x0)
	tM.logAnswer(event.Command.Query, answer, 0x0)
}
