package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"errors"
	"net/http"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

type LogLevel int64 

func (l *LogLevel) UnmarshalJSON(b []byte) error {
	var levelString string
	err := json.Unmarshal(b, &levelString)
	if err != nil {
		return err
	}

	switch levelString {
	case "EMERGENCY":
		*l = EMERGENCY
	case "CRITICAL":
		*l = CRITICAL
	case "ERROR":
		*l = ERROR
	case "ALERT":
		*l = ALERT
	case "WARNING":
		*l = WARNING
	case "NOTICE":
		*l = NOTICE
	case "INFORMATIONAL":
		*l = INFORMATIONAL
	case "DEBUG":
		*l = DEBUG
	default:
		return errors.New("invalid log level")
	}

	return nil
}

const (
	EMERGENCY LogLevel = iota
	CRITICAL LogLevel = iota
	ERROR LogLevel = iota
	ALERT LogLevel = iota
	WARNING LogLevel = iota
	NOTICE LogLevel = iota
	INFORMATIONAL LogLevel = iota
	DEBUG LogLevel = iota
)

type Log struct {
	Timestamp time.Time `json:"timestamp"`
	Correlation_id string `json:"correlation_id"`
	Severity LogLevel `json:"severity"`
	Message string `json:"message"`
}

func handleLog(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if r.URL.Path != "/api/v1/logs" {
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

func handleCreate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := context.Background()

	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200",
		},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	sample_log := Log{Timestamp: time.Now(), Correlation_id: req.Correlation_id, Severity: req.Severity, Message: req.Message}

	id, err := Insert(client, ctx, sample_log)
	if err != nil {
		log.Fatalf("Error inserting doc: %s", err)
	}

	response := createResponse{id.String()}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

	fmt.Println("Log Added")
	fmt.Println(id)

}

func handleGet(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	ctx := context.Background()

	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200",
		},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	doc, err := FindOne(client, ctx, p.ByName("id"))
	if err != nil {
		fmt.Printf("Error finding one: %s", err)
	}

	response := getResponse{
		Timestamp: doc.Timestamp,
		Correlation_id: doc.Correlation_id,
		Severity: doc.Severity,
		Message: doc.Message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

}


func Insert(client *elasticsearch.Client, ctx context.Context, post Log) (uuid.UUID, error) {
	id := uuid.New()

	body, err := json.Marshal(post)
	if err != nil {
		log.Fatalf("Error marshalling %w", err)
		return id, err
	}

	req := esapi.CreateRequest{
		Index: "logs",
		DocumentID: id.String(),
		Body: bytes.NewReader(body),
	}

	res, err := req.Do(ctx, client)
	if err != nil {
		return id, fmt.Errorf("insert: request: %w", err)
	}
	defer res.Body.Close()
 
	if res.StatusCode == 409 {
		return id, fmt.Errorf("insert 409: response: %s", res.String())
	}
 
	if res.IsError() {
		return id, fmt.Errorf("insert: response: %s", res.String())
	}

	bytes, err := io.ReadAll(res.Body)
	fmt.Println(string(bytes))

	return id, nil 
}

func FindOne(client *elasticsearch.Client, ctx context.Context, id string) (Log, error){
	req := esapi.GetRequest{
		Index: "logs",
		DocumentID: id,
	}

	res, err := req.Do(ctx, client)
	if err != nil {
		return Log{}, fmt.Errorf("get: request: %w", err)
	}
	defer res.Body.Close()
 
	if res.StatusCode == 404 {
		return Log{}, fmt.Errorf("get 404: response: %s", res.String())
	}
 
	if res.IsError() {
		return Log{}, fmt.Errorf("get: response: %s", res.String())
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return Log{}, fmt.Errorf("find one: decode: %w", err)
	}

	fmt.Println(string(bytes))

	log := Log{}
	err = json.Unmarshal(bytes, &log)
	if err != nil {
		fmt.Printf("err = %v\n", err)
	}

	return log, nil
}

func main() {

	ctx := context.Background()

	//errorlog := log.New(os.Stdout, "APP ", log.LstdFlags)
	var index_name []string;
	index_name = append(index_name, "logs")

	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://localhost:9200",
		},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	res, err := client.Info()
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}

	defer res.Body.Close()
	log.Println(res)

	res, err = client.Indices.Create("logs")
	if err != nil {
		log.Fatalf("Error Create Index: %s", err)
	}
	log.Println(res)

	sample_log := Log{Timestamp: time.Now(), Correlation_id: "admin_testing", Severity: INFORMATIONAL, Message: "This is a test log."}

	id, err := Insert(client, ctx, sample_log)
	if err != nil {
		log.Fatalf("Error inserting doc: %s", err)
	}

	_, err = FindOne(client, ctx, id.String())
	if err != nil {
		log.Fatalf("Error finding one: %s", err)
	}

	res, err = client.Indices.Delete(index_name)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	log.Println(res)

	//curl --request POST "localhost:8080/api/v1/logs" -d '{"correlation_id":"admin_test", "severity":"INFORMATIONAL", "message":"curl test log"}'

	router := httprouter.New()
	router.GET("/api/v1/logs/:id", handleGet)
	router.POST("/api/v1/logs", handleCreate)

	log.Fatal(http.ListenAndServe(":8080", router))
}









