package main

// MIT Licensed - see LICENSE

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/pschlump/godebug"
	"github.com/pschlump/json"
	"golang.org/x/crypto/pbkdf2"
)

// Exists return true if the file/path or directory exists
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

var invalidMode = errors.New("Invalid Mode")

// Fopen works like the C fopen - it opens a file based on mode and return sthe file
func Fopen(fn string, mode string) (file *os.File, err error) {
	file = nil
	if mode == "r" {
		file, err = os.Open(fn) // For read access.
	} else if mode == "w" {
		file, err = os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	} else if mode == "a" {
		file, err = os.OpenFile(fn, os.O_RDWR|os.O_APPEND, 0660)
		if err != nil {
			file, err = os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		}
	} else {
		err = invalidMode
	}
	return
}

// GetParam returns the named parameter from the URl or the defauilt value if not set.
func GetParam(www http.ResponseWriter, req *http.Request, name string, dflt string) (rv string) {
	found := false
	value := dflt

	method := req.Method
	if dbFlag["GetVal"] {
		fmt.Printf("GetVar name=%s req.Method %s AT:%s\n", name, method, godebug.LF())
	}
	if method == "POST" || method == "PUT" {
		if str := req.PostFormValue(name); str != "" {
			value = str
			found = true
		}
	} else if method == "GET" || method == "DELETE" {
		if dbFlag["GetVal"] {
			fmt.Printf("AT:%s\n", godebug.LF())
		}
		qq := req.URL.Query()
		strArr, ok := qq[name]
		if dbFlag["GetVal"] {
			fmt.Printf("AT:%s strArr = %s ok = %v\n", godebug.LF(), godebug.SVar(strArr), ok)
		}
		if ok {
			if dbFlag["GetVal"] {
				fmt.Printf("AT:%s\n", godebug.LF())
			}
			if len(strArr) > 0 {
				if dbFlag["GetVal"] {
					fmt.Printf("AT:%s\n", godebug.LF())
				}
				value = strArr[0]
				found = true
			} else {
				if dbFlag["GetVal"] {
					fmt.Printf("AT:%s\n", godebug.LF())
				}
				fmt.Fprintf(os.Stderr, "Multiple values for [%s]\n", name)
				found = false
			}
		}
	}
	if found {
		rv = value
	}
	return
}

// SVar marshals data into a JSON string
func SVar(v interface{}) string {
	s, err := json.Marshal(v)
	// s, err := json.MarshalIndent ( v, "", "\t" )
	if err != nil {
		return fmt.Sprintf("Error:%s", err)
	} else {
		return string(s)
	}
}

// SVarI marshals data into a JSON string with tab indentation
func SVarI(v interface{}) string {
	// s, err := json.Marshal ( v )
	s, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return fmt.Sprintf("Error:%s", err)
	} else {
		return string(s)
	}
}

// CheckAuth Checks a X-Auth header authentication to validate a user.  It returns true if
// the user is authorized, else it creates a 401 error and returns.
func CheckAuth(www http.ResponseWriter, req *http.Request) (ok bool) {
	token := req.Header.Get("X-Auth")
	if token == "" {
		AnError(www, req, 401, "Login required")
		return false
	}

	key := fmt.Sprintf("qr-token:%s", token)

	str, err := redisClient.Cmd("GET", key).Str()
	if err != nil || str == "" {
		AnError(www, req, 401, "Login required")
		return false
	}

	return true
}

// AnError reports an error and logs the error to stderr.
func AnError(www http.ResponseWriter, req *http.Request, httpStatus int, msg string) {
	fmt.Fprintf(os.Stderr, "Error: uri=%s status=%d msg=%s at:%s\n", req.RequestURI, httpStatus, msg, godebug.LF(-4))
	fmt.Fprintf(logFile, "Error: uri=%s status=%d msg=%s at:%s\n", req.RequestURI, httpStatus, msg, godebug.LF(-4))
	http.Error(www, fmt.Sprintf("Error: %s\n", msg), httpStatus)
}

// CreateUser will create a user in Redis with a qr-auth: and qr-salt: keys.
func CreateUser(un, pw string) (err error) {

	RanV, _ := GenRandNumber(12)
	salt := fmt.Sprintf("%x", RanV)

	pwHash := fmt.Sprintf("%x", pbkdf2.Key([]byte(pw), []byte(salt), NIterations, 64, sha256.New))
	key := fmt.Sprintf("qr-auth:%s", un)

	err = redisClient.Cmd("SET", key, pwHash).Err
	if err != nil {
		return fmt.Errorf("Unable to set user authenication: %s", err)
	}

	key = fmt.Sprintf("qr-salt:%s", un)
	err = redisClient.Cmd("SET", key, salt).Err
	if err != nil {
		return fmt.Errorf("Unable to set user authenication/salt: %s", err)
	}

	return nil
}

// CheckSetup checks to see if the Redis database has been initialized.  If not -then it creates
// the necessary keys init.
func CheckSetup() {
	key := fmt.Sprintf("qr-id:")
	str, err := redisClient.Cmd("GET", key).Str()
	if err != nil || str == "" {
		fmt.Printf("Setting Up Redis\n")
		// Setup the Redis Database
		err = redisClient.Cmd("SET", key, "10000").Err
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting up Redis: %s\n", err)
			os.Exit(3)
		}
	}
}

/* vim: set noai ts=4 sw=4: */
