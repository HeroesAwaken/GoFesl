package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/HeroesAwaken/GoAwaken/core"
	"github.com/go-redis/redis"

	"net/http"
	"net/http/pprof"

	log "github.com/HeroesAwaken/GoAwaken/Log"
)

var (
	// BuildTime of the build provided by the build command
	BuildTime = "Not provided"
	// GitHash of build provided by the build command
	GitHash = "Not provided"
	// GitBranch of the build provided by the build command
	GitBranch = "Not provided"
	// compileVersion we are receiving by the build command
	CompileVersion = "0"
	// Version of the Application
	Version = "0.0.2"

	// MyConfig Default configuration
	MyConfig = Config{
		MysqlServer: "localhost:3306",
		MysqlUser:   "loginserver",
		MysqlDb:     "loginserver",
		MysqlPw:     "",
	}

	mem runtime.MemStats
)

func emtpyHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugln(r.URL.String())
	fmt.Fprintf(w, "<update><status>Online</status></update>")
}

func relationship(w http.ResponseWriter, r *http.Request) {
	log.Noteln(r.URL.String())
	log.Noteln("<update><status>Online</status><name>Test</name></update>")
	fmt.Fprintf(w, "<update><id>1</id><name>Test</name><state>ACTIVE</state><type>server</type><status>Online</status><realid>123</realid></update>")
}

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	serverKey := r.Header.Get("X-SERVER-KEY")
	if serverKey != "" {
		log.Noteln("Server " + serverKey + " authenticating.")
		fmt.Fprintf(w, "<success><token>"+serverKey+"</token></success>")
	} else {
		userKey, err := r.Cookie("magma")
		if err != nil {
		}
		log.Noteln("<success><token code=\"NEW_TOKEN\">" + userKey.Value + "</token></success>")
		fmt.Fprintf(w, "<success><token code=\"NEW_TOKEN\">"+userKey.Value+"</token></success>")
	}
}

func main() {
	var (
		configPath   = flag.String("config", "config.yml", "Path to yml configuration file")
		logLevel     = flag.String("logLevel", "error", "LogLevel [error|warning|note|debug]")
		certFileFlag = flag.String("cert", "cert.pem", "[HTTPS] Location of your certification file. Env: LOUIS_HTTPS_CERT")
		keyFileFlag  = flag.String("key", "key.pem", "[HTTPS] Location of your private key file. Env: LOUIS_HTTPS_KEY")
	)
	flag.Parse()

	if CompileVersion != "0" {
		Version = Version + "." + CompileVersion
	}

	log.SetLevel(*logLevel)
	log.Notef("Starting up v%s - %s %s %s", Version, BuildTime, GitBranch, GitHash)

	MyConfig.Load(*configPath)

	r := http.NewServeMux()

	// Register pprof handlers
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.HandleFunc("/nucleus/authToken", sessionHandler)
	r.HandleFunc("/relationships/roster/nucleus:1817675496", relationship)
	r.HandleFunc("/", emtpyHandler)

	go func() {
		log.Noteln(http.ListenAndServe("0.0.0.0:80", r))
	}()
	go func() {
		log.Noteln(http.ListenAndServeTLS("0.0.0.0:443", *certFileFlag, *keyFileFlag, r))
	}()
	// Startup done

	// DB Connection
	dbConnection := new(core.DB)
	dbSQL, err := dbConnection.New(MyConfig.MysqlServer, MyConfig.MysqlDb, MyConfig.MysqlUser, MyConfig.MysqlPw)
	if err != nil {
		log.Fatalln("Error connecting to DB:", err)
	}

	// Redis Connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     MyConfig.RedisServer,
		Password: MyConfig.RedisPassword,
		DB:       MyConfig.RedisDB,
	})
	_, err = redisClient.Ping().Result()
	if err != nil {
		log.Fatalln("Error connecting to redis:", err)
	}

	feslManager := new(FeslManager)
	feslManager.New("FM", "18270", *certFileFlag, *keyFileFlag, false, dbSQL, redisClient)
	serverManager := new(FeslManager)
	serverManager.New("SFM", "18051", *certFileFlag, *keyFileFlag, true, dbSQL, redisClient)

	theaterManager := new(TheaterManager)
	theaterManager.New("TM", "18275", dbSQL, redisClient)
	servertheaterManager := new(TheaterManager)
	servertheaterManager.New("STM", "18056", dbSQL, redisClient)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		log.Noteln("Captured" + sig.String() + ". Shutting down.")
		os.Exit(0)
	}
}
