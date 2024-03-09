package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	 "github.com/google/uuid"
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
	Timestamp string `json:"timestamp"`
	Correlation_id string `json:"correlation_id"`
	Severity LogLevel `json:"severity"`
	Message string `json:"message"`
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

	return id, nil 
}

func FindOne(client *elasticsearch.Client, ctx context.Context, id uuid.UUID) error {
	req := esapi.GetRequest{
		Index: "logs",
		DocumentID: id.String(),
	}

	res, err := req.Do(ctx, client)
	if err != nil {
		return fmt.Errorf("get: request: %w", err)
	}
	defer res.Body.Close()
 
	if res.StatusCode == 404 {
		return fmt.Errorf("get 404: response: %s", res.String())
	}
 
	if res.IsError() {
		return fmt.Errorf("get: response: %s", res.String())
	}

	var m Log;

	if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
		return fmt.Errorf("find one: decode: %w", err)
	}

	log.Println(m)

	return nil
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

	//res, err = client.Indices.Create("logs")
	//if err != nil {
	//	log.Fatalf("Error Create Index: %s", err)
	//}
	//log.Println(res)


	sample_log := Log{Timestamp: time.Now().String(), Correlation_id: "admin_testing", Severity: INFORMATIONAL, Message: "This is a test log."}

	id, err := Insert(client, ctx, sample_log)
	if err != nil {
		log.Fatalf("Error inserting doc: %s", err)
	}

	err = FindOne(client, ctx, id)
	if err != nil {
		log.Fatalf("Error finding one: %s", err)
	}

	res, err = client.Indices.Delete(index_name)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	log.Println(res)
	

	http.HandleFunc("/", handleLog)
	http.ListenAndServe(":8081", nil)
}
