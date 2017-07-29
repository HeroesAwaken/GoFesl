package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/HeroesAwaken/GoAwaken/core"
	"github.com/SpencerSharkey/GoFesl/fesl"
	"github.com/SpencerSharkey/GoFesl/log"
	"github.com/SpencerSharkey/GoFesl/theater"
	"github.com/go-redis/redis"

	"net/http"
	"net/http/pprof"
)

// Initialize flag-parameters and config
func init() {
	flag.StringVar(&configPath, "config", "config.yml", "Path to yml configuration file")
	flag.StringVar(&logLevel, "logLevel", "error", "LogLevel [error|warning|note|debug]")
	flag.StringVar(&certFileFlag, "cert", "cert.pem", "[HTTPS] Location of your certification file. Env: LOUIS_HTTPS_CERT")
	flag.StringVar(&keyFileFlag, "key", "key.pem", "[HTTPS] Location of your private key file. Env: LOUIS_HTTPS_KEY")
	flag.Parse()

	log.SetLevel(logLevel)
	MyConfig.Load(configPath)

	if CompileVersion != "0" {
		Version = Version + "." + CompileVersion
	}
}

var (
	configPath   string
	logLevel     string
	certFileFlag string
	keyFileFlag  string

	// CompileVersion we are receiving by the build command
	CompileVersion = "0"
	// Version of the Application
	Version = "0.0.6"

	// MyConfig Default configuration
	MyConfig = Config{
		MysqlServer: "localhost:3306",
		MysqlUser:   "loginserver",
		MysqlDb:     "loginserver",
		MysqlPw:     "",
	}

	mem runtime.MemStats

	AppName = "HeroesServer"
)

func emtpyHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugln(r.URL.String())
	fmt.Fprintf(w, "<update><status>Online</status></update>")
}

func relationship(w http.ResponseWriter, r *http.Request) {
	log.Noteln(r.URL.String())
	log.Noteln("<update><id>1</id><name>Test</name><state>ACTIVE</state><type>server</type><status>Online</status><realid>1817224672</realid></update>")
	fmt.Fprintf(w, "<update><id>1</id><name>Test</name><state>ACTIVE</state><type>server</type><status>Online</status><realid>1817224672</realid></update>")
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

func collectGlobalMetrics(iDB *core.InfluxDB) {
	runtime.ReadMemStats(&mem)
	tags := map[string]string{"metric": "server_metrics", "server": "global"}
	fields := map[string]interface{}{
		"memAlloc":      int(mem.Alloc),
		"memTotalAlloc": int(mem.TotalAlloc),
		"memHeapAlloc":  int(mem.HeapAlloc),
		"memHeapSys":    int(mem.HeapSys),
	}

	iDB.AddMetric("server_metrics", tags, fields)
	iDB.Flush()
}

func main() {
	log.Notef("Starting up v%s", Version)

	r := http.NewServeMux()

	// Register pprof handlers
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.HandleFunc("/nucleus/authToken", sessionHandler)
	r.HandleFunc("/relationships/roster/server:7eb6155c-ac70-4567-9fc4-732d56a9334a", relationship)
	r.HandleFunc("/relationships/roster/nucleus:158", relationship)
	r.HandleFunc("/relationships/roster/nucleus:1817224672", relationship)
	r.HandleFunc("/", emtpyHandler)

	go func() {
		log.Noteln(http.ListenAndServe("0.0.0.0:80", r))
	}()
	go func() {
		log.Noteln(http.ListenAndServeTLS("0.0.0.0:443", certFileFlag, keyFileFlag, r))
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

	// Influx Connection
	metricConnection := new(core.InfluxDB)
	err = metricConnection.New(MyConfig.InfluxDBHost, MyConfig.InfluxDBDatabase, MyConfig.InfluxDBUser, MyConfig.InfluxDBPassword, AppName, Version)
	if err != nil {
		log.Fatalln("Error connecting to MetricsDB:", err)
	}

	globalMetrics := time.NewTicker(time.Second * 10)
	go func() {
		for range globalMetrics.C {
			collectGlobalMetrics(metricConnection)
		}
	}()

	feslManager := new(fesl.FeslManager)
	feslManager.New("FM", "18270", certFileFlag, keyFileFlag, false, dbSQL, redisClient, metricConnection)
	serverManager := new(fesl.FeslManager)
	serverManager.New("SFM", "18051", certFileFlag, keyFileFlag, true, dbSQL, redisClient, metricConnection)

	theaterManager := new(theater.TheaterManager)
	theaterManager.New("TM", "18275", dbSQL, redisClient, metricConnection)
	servertheaterManager := new(theater.TheaterManager)
	servertheaterManager.New("STM", "18056", dbSQL, redisClient, metricConnection)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		log.Noteln("Captured" + sig.String() + ". Shutting down.")
		os.Exit(0)
	}
}
