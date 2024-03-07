package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type LogLevel int64 

const (
	EMERGENCY LogLevel = iota
	ALERT LogLevel = iota
	CRITICAL LogLevel = iota
	ERROR LogLevel = iota
	WARNING LogLevel = iota
	NOTICE LogLevel = iota
	INFORMATIONAL LogLevel = iota
	DEBUG LogLevel = iota
)

type Log struct {
	timestamp string
	correlation_id string
	log_level LogLevel
	message string
}

func handleLog(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		defer r.Body.Close()

		if err != nil {
			fmt.Printf("An error has occurred reading body")
			return
		}

		fmt.Printf("Headers: %+v\n", r.Header)

		if len(bodyBytes) == 0 {
			fmt.Printf("No body supplied")
			return
		}

		var prettyJSON bytes.Buffer

		if err = json.Indent(&prettyJSON, bodyBytes, "", "\t"); err != nil {
			fmt.Printf("JSON parse error: %v", err)	
			return
		}

		fmt.Println(string(prettyJSON.Bytes()))
	}

}

func main() {
	http.HandleFunc("/", handleLog)
	http.ListenAndServe(":8081", nil)
}
