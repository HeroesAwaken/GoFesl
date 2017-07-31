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
	"github.com/gorilla/mux"

	"net/http"
	"net/http/pprof"
)

// Initialize flag-parameters and config
func init() {
	flag.StringVar(&configPath, "config", "config.yml", "Path to yml configuration file")
	flag.StringVar(&logLevel, "logLevel", "error", "LogLevel [error|warning|note|debug]")
	flag.StringVar(&certFileFlag, "cert", "cert.pem", "[HTTPS] Location of your certification file. Env: LOUIS_HTTPS_CERT")
	flag.StringVar(&keyFileFlag, "key", "key.pem", "[HTTPS] Location of your private key file. Env: LOUIS_HTTPS_KEY")
	flag.BoolVar(&localMode, "localMode", false, "Use in local modus")

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
	localMode    bool

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
	vars := mux.Vars(r)
	log.Noteln(r.URL.String())
	log.Noteln("<update><id>1</id><name>Test</name><state>ACTIVE</state><type>server</type><status>Online</status><realid>" + vars["id"] + "</realid></update>")
	fmt.Fprintf(w, "<update><id>1</id><name>Test</name><state>ACTIVE</state><type>server</type><status>Online</status><realid>"+vars["id"]+"</realid></update>")
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

func entitlementsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Noteln("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\" ?><entitlements><entitlement><entitlementId>1</entitlementId><entitlementTag>WEST_Custom_Item_142</entitlementTag><status>ACTIVE</status><userId>2</userId></entitlement></entitlements>")
	fmt.Fprintf(w,
		"<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\" ?>"+
			"<entitlements>"+
			"	<entitlement>"+
			"		<entitlementId>1</entitlementId>"+
			"		<entitlementTag>WEST_Custom_Item_142</entitlementTag>"+
			"		<status>ACTIVE</status>"+
			"		<userId>"+vars["heroID"]+"</userId>"+
			"	</entitlement>"+
			"	<entitlement>"+
			"		<entitlementId>1253</entitlementId>"+
			"		<entitlementTag>WEST_Custom_Item_142</entitlementTag>"+
			"		<status>ACTIVE</status>"+
			"		<userId>"+vars["heroID"]+"</userId>"+
			"	</entitlement>"+
			"</entitlements>")
}

func offersHandler(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	log.Noteln("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\" ?><products></products>")
	fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\" ?><products><product></product></products>")

}

func walletsHandler(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	log.Noteln("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\" ?><billingAccounts></billingAccounts>")
	fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\" ?><billingAccounts><walletAccount><currency>hp</currency><balance>1</balance></billingAccounts>")

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

	r := mux.NewRouter()

	// Register pprof handlers
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	r.HandleFunc("/nucleus/authToken", sessionHandler)
	r.HandleFunc("/relationships/roster/{type}:{id}", relationship)

	r.HandleFunc("/nucleus/entitlements/{heroID}", entitlementsHandler)
	r.HandleFunc("/nucleus/wallets/{heroID}", walletsHandler)
	r.HandleFunc("/ofb/products", offersHandler)

	r.HandleFunc("/", emtpyHandler)

	if localMode {
		go func() {
			log.Noteln(http.ListenAndServe("0.0.0.0:80", r))
		}()
		go func() {
			log.Noteln(http.ListenAndServeTLS("0.0.0.0:443", certFileFlag, keyFileFlag, r))
		}()
	} else {

		go func() {
			log.Noteln(http.ListenAndServe("0.0.0.0:8080", r))
		}()
		go func() {
			log.Noteln(http.ListenAndServeTLS("0.0.0.0:8443", certFileFlag, keyFileFlag, r))
		}()
	}
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
	feslManager.New("FM", "18270", certFileFlag, keyFileFlag, false, dbSQL, redisClient, metricConnection, localMode)
	serverManager := new(fesl.FeslManager)
	serverManager.New("SFM", "18051", certFileFlag, keyFileFlag, true, dbSQL, redisClient, metricConnection, localMode)

	theaterManager := new(theater.TheaterManager)
	theaterManager.New("TM", "18275", dbSQL, redisClient, metricConnection, localMode)
	servertheaterManager := new(theater.TheaterManager)
	servertheaterManager.New("STM", "18056", dbSQL, redisClient, metricConnection, localMode)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		log.Noteln("Captured" + sig.String() + ". Shutting down.")
		os.Exit(0)
	}
}
