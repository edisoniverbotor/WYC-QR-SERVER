package main

// MIT Licensed - see LICENSE

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pschlump/godebug"
	"github.com/pschlump/qr-svr/ReadConfig"
	"github.com/pschlump/radix.v2/redis"
	"github.com/pschlump/uuid"
	"golang.org/x/crypto/pbkdf2"
)

type ConfigType struct {

	// Redis connection stuff
	RedisConnectHost string `json:"redis_host" default:"$ENV$REDIS_HOST"`
	RedisConnectAuth string `json:"redis_auth" default:"$ENV$REDIS_AUTH"`
	RedisConnectPort string `json:"redis_port" default:"6379"`

	// server config
	HostPort string `json:"host_port" default:"localhost:8333"` //
	Dir      string `json:"dir" default:"./www"`                // Directory for serve of files
	QRDir    string `json:"dir" default:"./www/q"`              // Directory for writing images to
	QRUri    string `json:"dir" default:"./q"`                  // URL path for serving QRs
	LogFile  string `json:"log_file" default:"./log/log.out"`   //

	// Login Duration
	LoginTTL int `json:"session_persistence" default:"2592000"` // 30 days * # of sec per day ( 60 * 60 * 24 )

	// QR config stuff
	Level  string `json:"qr_level" default:"H"`  // Redundancy level in QR
	QRSize int    `json:"qr_size" default:"256"` // Pixel size of image
}

var gCfg ConfigType

var optCfg = flag.String("cfg", "cfg.json", "config file for this call")
var optHostPort = flag.String("hostport", "", "Host/Port to listen on")
var optDir = flag.String("dir", "", "Directory to server from")
var optCreateUser = flag.String("create-user", "", "Username to create (must also have --password)")
var optPassword = flag.String("password", "", "Password to go with username")

var dbFlag map[string]bool
var NIterations = 50000 // # of iterations of hashing for passwords
var redisClient *redis.Client
var logFile *os.File

func init() {
	dbFlag = make(map[string]bool)
	logFile = os.Stderr
}

// repHandlerStatus returns a status that show if the server is connected and live.
func respHandlerStatus(www http.ResponseWriter, req *http.Request) {
	q := req.RequestURI

	var rv, rok string
	rok = "Redis OK"

	// check redis connection
	_, err := redisClient.Cmd("GET", "qr-id:").Int()
	if err != nil {
		rok = "Failed to connect to Redis"
		return
	}

	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	rv = fmt.Sprintf(`{"status":"success","name":"qr-svr version 1.0.0","URI":%q,"req":%s, "response_header":%s, "redis":%q}`, q, SVarI(req), SVarI(www.Header()), rok)

	io.WriteString(www, rv)
}

/*
/api/count?id=ID
*/
func respHandlerCount(www http.ResponseWriter, req *http.Request) {
	if !CheckAuth(www, req) {
		return
	}

	id := GetParam(www, req, "id", "")
	if id == "" {
		AnError(www, req, 406, "Missing Parameter")
		return
	}

	key := fmt.Sprintf("qr-count:%s", id)
	n, err := redisClient.Cmd("GET", key).Str()
	if err != nil {
		AnError(www, req, 500, "Config Error 0")
		return
	}

	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(www, `{"status":"success","count":%q}`+"\n", n)
}

/*
/api/upd-qr?id=ID&url=XXX
*/
func respHandlerUpdQR(www http.ResponseWriter, req *http.Request) {
	if !CheckAuth(www, req) {
		return
	}

	id := GetParam(www, req, "id", "")
	if id == "" {
		AnError(www, req, 406, "Missing Parameter")
		return
	}
	xurl := GetParam(www, req, "url", "")
	if xurl == "" {
		AnError(www, req, 406, "Missing Parameter")
		return
	}

	key := fmt.Sprintf("qrr:%s", id)

	err := redisClient.Cmd("SET", key, xurl).Err
	if err != nil {
		AnError(www, req, 500, "Config Error 5")
		return
	}

	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(www, `{"status":"success"}`)
}

/*
/api/gen-qr?url=XXX - initial XXX url to set ID to, returns QR and ID as JSON
*/
func respHandlerGenQR(www http.ResponseWriter, req *http.Request) {
	if !CheckAuth(www, req) {
		return
	}

	// fmt.Printf("AT: %s\n", godebug.LF())
	xurl := GetParam(www, req, "url", "")
	if xurl == "" {
		xurl = fmt.Sprintf("http://%s/404page.html", gCfg.HostPort)
	}

	key := fmt.Sprintf("qr-id:")

	// fmt.Printf("AT: %s\n", godebug.LF())
	// get the ID / Increment it.
	id, err := redisClient.Cmd("INCR", key).Int()
	if err != nil {
		AnError(www, req, 500, fmt.Sprintf("Config Error 1: %s at:%s", err, godebug.LF()))
		return
	}

	// fmt.Printf("AT: %s\n", godebug.LF())
	// GenQR call - to generate image and save it.
	uri, _ /*pth*/, err := GenQR(gCfg.QRDir, gCfg.QRUri, gCfg.HostPort, fmt.Sprintf("%d", id))
	if err != nil {
		AnError(www, req, 500, "Config Error 2")
		return
	}

	// fmt.Printf("AT: %s\n", godebug.LF())
	// set in Redis
	key = fmt.Sprintf("qrr:%d", id)
	err = redisClient.Cmd("SET", key, xurl).Err
	if err != nil {
		AnError(www, req, 500, "Config Error 3")
		return
	}
	// fmt.Printf("AT: %s\n", godebug.LF())
	key = fmt.Sprintf("qr-count:%d", id)
	err = redisClient.Cmd("SET", key, "0").Err
	if err != nil {
		AnError(www, req, 500, "Config Error 4")
		return
	}

	img := fmt.Sprintf("http://%s/%s/%d.png", gCfg.HostPort, gCfg.QRUri, id)

	// fmt.Printf("AT: %s\n", godebug.LF())
	// generate JSON response w/ ID and QR
	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	// fmt.Fprintf(www, `{"status":"success", "id":"%d", "url":%q, "qr_url":%q }`, id, xurl, uri)
	fmt.Fprintf(www, `{"status":"success", "id":"%d", "url":%q, "qr_url":%q, "qr_encoded":%q }`, id, xurl, img, uri)

}

/*
/api/get-jwt?un=X&pw=Y -> token
	If missing un/pw params then 406
	Generate token if valid - else 401
	Lookup in redis qr-salt:X to get per-user salt
	Lookup in Redis qr-auth:X -> hash(salt:password) - compare to hash(salt:password)
*/
func respHandlerGetAuth(www http.ResponseWriter, req *http.Request) {

	// fmt.Printf("AT: %s\n", godebug.LF())
	un := GetParam(www, req, "un", "")
	if un == "" {
		AnError(www, req, 406, "Missing Username")
		return
	}
	pw := GetParam(www, req, "pw", "")
	if pw == "" {
		AnError(www, req, 406, "Missing Password")
		return
	}

	// fmt.Printf("AT: %s\n", godebug.LF())
	key := fmt.Sprintf("qr-auth:%s", un)

	pwHash, err := redisClient.Cmd("GET", key).Str()
	if err != nil {
		AnError(www, req, 401, "Not Found")
		return
	}

	key = fmt.Sprintf("qr-salt:%s", un)

	// fmt.Printf("AT: %s\n", godebug.LF())
	salt, err := redisClient.Cmd("GET", key).Str()
	if err != nil {
		AnError(www, req, 401, "Not Found")
		return
	}

	dk := fmt.Sprintf("%x", pbkdf2.Key([]byte(pw), []byte(salt), NIterations, 64, sha256.New))

	if pwHash != dk {
		AnError(www, req, 401, "Not Found")
		return
	}

	// fmt.Printf("AT: %s\n", godebug.LF())
	newUUID, err := uuid.NewV4()
	if err != nil {
		AnError(www, req, http.StatusInternalServerError, "Not Found") // 500
		return
	}
	token := newUUID.String()

	key = fmt.Sprintf("qr-token:%s", token)

	// fmt.Printf("AT: %s\n", godebug.LF())
	err = redisClient.Cmd("SETEX", key, gCfg.LoginTTL, "yes").Err
	if err != nil {
		AnError(www, req, 500, "Config Error 8")
		return
	}

	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(www, `{"status":"success","auth_token":%q}`+"\n", token)

}

/*
/q/{ID}
*/
func respHandlerRedirect(www http.ResponseWriter, req *http.Request) {
	if len(req.RequestURI[3:]) <= 3 {
		AnError(www, req, 404, "Not Found")
		return
	}
	// fmt.Printf("AT: %s URi ->%s<\n", godebug.LF(), req.RequestURI)

	id := req.RequestURI[3:]
	// fmt.Printf("AT: %s id ->%s<\n", godebug.LF(), id)

	key := fmt.Sprintf("qrr:%s", id)
	to, err := redisClient.Cmd("GET", key).Str()
	if err != nil {
		AnError(www, req, 404, "Not Found")
		return
	}

	// fmt.Printf("AT: %s\n", godebug.LF())
	key = fmt.Sprintf("qr-count:%s", id)
	redisClient.Cmd("INCR", key)

	// fmt.Printf("AT: %s\n", godebug.LF())
	h := www.Header()
	h.Set("Location", HexEscapeNonASCII(to))
	www.WriteHeader(303)

	fmt.Fprintf(www, `<center><a href="%s">Redirect To: %s</a></center>\n`, to, to)
}

// Returns a status on valid/invalid tokens - used for testing.
func respHandlerAuthTokenValid(www http.ResponseWriter, req *http.Request) {
	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !CheckAuth(www, req) {
		fmt.Fprintf(www, `{"status":"invalid"}`+"\n")
	} else {
		fmt.Fprintf(www, `{"status":"success","msg":"token is valid"}`+"\n")
	}
}

// Same a /q/ but w/o the actual Redirect (HTTP 203)
func respHandlerLookup(www http.ResponseWriter, req *http.Request) {
	id := GetParam(www, req, "id", "")
	if id == "" {
		AnError(www, req, 406, "Missing Parameter")
		return
	}

	key := fmt.Sprintf("qrr:%s", id)

	to, err := redisClient.Cmd("GET", key).Str()
	if err != nil {
		AnError(www, req, 404, "Not Found")
		return
	}

	to = strings.Replace(to, "/./", "/", -1)
	uri := fmt.Sprintf("http://%s/%s/%s.png", gCfg.HostPort, gCfg.QRUri, id)
	uri = strings.Replace(uri, "/./", "/", -1)

	www.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(www, `{"status":"success", "url":%q, "qr":%q}`+"\n", to, uri)
}

func main() {

	// ------------------------------------------------------------------------------
	// Parse CLI
	// ------------------------------------------------------------------------------
	flag.Parse()

	fns := flag.Args()
	if len(fns) != 0 {
		fmt.Fprintf(os.Stderr, "Error: extra argument supplied\n")
		os.Exit(1)
	}

	if optCfg == nil {
		fmt.Printf("--cfg is a required parameter\n")
		os.Exit(1)
	}

	sCfg := *optCfg
	if x := os.Getenv(sCfg); x != "" {
		sCfg = x
	}

	// ------------------------------------------------------------------------------
	// Read in Configuraiton
	// ------------------------------------------------------------------------------
	err := ReadConfig.ReadFile(sCfg, &gCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read configuration: %s error %s\n", sCfg, err)
		os.Exit(1)
	}

	// ------------------------------------------------------------------------------
	// Connect to Redis
	// ------------------------------------------------------------------------------
	var ok bool
	redisClient, ok = RedisClient(dbFlag, &gCfg)
	if !ok {
		fmt.Fprintf(os.Stderr, "Unable to connect to Redis\n")
		os.Exit(1)
	}

	// ------------------------------------------------------------------------------
	// Setup
	// ------------------------------------------------------------------------------
	CheckSetup() // Check that Redis is setup - if not - then create keys.

	if *optCreateUser != "" { // See if we are running at the command line to create a user.
		if *optPassword == "" {
			fmt.Fprintf(os.Stderr, "Must supply both a --create-user [username] and --passwrod [password]\n")
			os.Exit(2)
		}

		err := CreateUser(*optCreateUser, *optPassword)
		if err == nil {
			fmt.Printf("User created: %s\n", *optCreateUser)
		} else {
			fmt.Printf("Error: %s\n", err)
		}

		os.Exit(0)
	}

	if *optHostPort != "" {
		gCfg.HostPort = *optHostPort
	}
	if *optDir != "" {
		gCfg.Dir = *optDir
	}

	if !Exists(gCfg.QRDir) {
		os.MkdirAll(gCfg.QRDir, 0755)
	}
	if !Exists(filepath.Dir(gCfg.LogFile)) {
		os.MkdirAll(filepath.Dir(gCfg.LogFile), 0755)
	}
	logFile, err = Fopen(gCfg.LogFile, "a")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to open log file >%s< error:%s\n", gCfg.LogFile, err)
		os.Exit(1)
	}

	// ------------------------------------------------------------------------------
	// URI Paths - Mux
	// ------------------------------------------------------------------------------
	http.HandleFunc("/api/status", respHandlerStatus)
	http.HandleFunc("/api/count", respHandlerCount)
	http.HandleFunc("/api/upd-qr", respHandlerUpdQR)
	http.HandleFunc("/api/get-qr", respHandlerGenQR)
	http.HandleFunc("/api/gen-qr", respHandlerGenQR)
	http.HandleFunc("/api/get-auth", respHandlerGetAuth)
	http.HandleFunc("/api/lookup", respHandlerLookup)
	http.HandleFunc("/api/auth-token-valid", respHandlerAuthTokenValid)
	http.HandleFunc("/Q/", respHandlerRedirect)
	http.Handle("/", http.FileServer(http.Dir(gCfg.Dir)))

	// ------------------------------------------------------------------------------
	// Run Server
	// ------------------------------------------------------------------------------
	log.Fatal(http.ListenAndServe(gCfg.HostPort, nil))
}

/* vim: set noai ts=4 sw=4: */
