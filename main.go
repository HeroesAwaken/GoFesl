package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"

	"net/http"
	"net/http/pprof"

	log "github.com/ReviveNetwork/GoRevive/Log"
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

	go func() {
		log.Noteln(http.ListenAndServe("0.0.0.0:6060", r))
	}()
	// Startup done

	feslManager := new(FeslManager)
	feslManager.New("FM", "18270", *certFileFlag, *keyFileFlag, false)

	serverManager := new(FeslManager)
	serverManager.New("SFM", "18051", *certFileFlag, *keyFileFlag, true)

	theaterManager := new(TheaterManager)
	theaterManager.New("TM", "18275")

	servertheaterManager := new(TheaterManager)
	servertheaterManager.New("STM", "18056")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		log.Noteln("Captured" + sig.String() + ". Shutting down.")
		os.Exit(0)
	}
}
