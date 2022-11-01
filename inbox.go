//go:build ignore

package main

import (
	"context"
	_ "embed"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rs/cors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Config struct {
	Port  int
	Host  string
	DBUrl string
}

type Message struct {
	Sender    string                 `json:"sender"`
	Receivers []string               `json:"receivers"`
	Content   map[string]interface{} `json:"content"`
}

type Storage interface {
	send(Message) (int, error)
	read(string, bool) ([]ReadAll, error)
	set(string, int, bool) error
}

type Inbox struct {
	storage Storage
}

//go:embed zenflows-crypto/src/verify_graphql.zen
var VERIFY string

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func (inbox *Inbox) sendHandler(w http.ResponseWriter, r *http.Request) {
	// Setup json response
	w.Header().Set("Content-Type", "application/json")
	enableCors(&w)
	result := map[string]interface{}{
		"success": false,
	}
	defer json.NewEncoder(w).Encode(result)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		result["success"] = false
		result["error"] = "Could not read the body of the request"
		return
	}
	zenroomData := ZenroomData{
		Gql:            b64.StdEncoding.EncodeToString(body),
		EdDSASignature: r.Header.Get("zenflows-sign"),
	}

	// Read a message object, I need the receivers
	var message Message
	err = json.Unmarshal(body, &message)
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}

	if len(message.Receivers) == 0 {
		result["success"] = false
		result["error"] = "No receivers"
		return
	}

	if len(message.Content) == 0 {
		result["success"] = false
		result["error"] = "Empty content"
		return
	}
	zenroomData.requestPublicKey(message.Sender)
	err = zenroomData.isAuth()
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}

	// For each receiver put the message in the inbox
	count, err := inbox.storage.send(message)
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}
	result["success"] = true
	result["count"] = count
	return
}

type ReadMessages struct {
	RequestId  int    `json:"request_id"`
	Receiver   string `json:"receiver"`
	OnlyUnread bool   `json:"only_unread"`
}

func (inbox *Inbox) readHandler(w http.ResponseWriter, r *http.Request) {
	// Setup json response
	w.Header().Set("Content-Type", "application/json")
	enableCors(&w)
	result := map[string]interface{}{
		"success": false,
	}
	defer json.NewEncoder(w).Encode(result)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}

	// Verify signature request
	zenroomData := ZenroomData{
		Gql:            b64.StdEncoding.EncodeToString(body),
		EdDSASignature: r.Header.Get("zenflows-sign"),
	}
	var readMessage ReadMessages
	err = json.Unmarshal(body, &readMessage)
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}
	zenroomData.requestPublicKey(readMessage.Receiver)
	err = zenroomData.isAuth()
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}
	messages, err := inbox.storage.read(readMessage.Receiver, readMessage.OnlyUnread)
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}

	result["success"] = true
	result["request_id"] = readMessage.RequestId
	result["messages"] = messages
	return
}

type SetMessage struct {
	MessageId int    `json:"message_id"`
	Receiver  string `json:"receiver"`
	Read      bool   `json:"read"`
}

func (inbox *Inbox) setHandler(w http.ResponseWriter, r *http.Request) {
	// Setup json response
	w.Header().Set("Content-Type", "application/json")
	enableCors(&w)
	result := map[string]interface{}{
		"success": false,
	}
	defer json.NewEncoder(w).Encode(result)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}

	// Verify signature request
	zenroomData := ZenroomData{
		Gql:            b64.StdEncoding.EncodeToString(body),
		EdDSASignature: r.Header.Get("zenflows-sign"),
	}
	var setMessage SetMessage
	err = json.Unmarshal(body, &setMessage)
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}
	zenroomData.requestPublicKey(setMessage.Receiver)
	err = zenroomData.isAuth()
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}
	err = inbox.storage.set(setMessage.Receiver, setMessage.MessageId, setMessage.Read)
	if err != nil {
		result["success"] = false
		result["error"] = err.Error()
		return
	}

	result["success"] = true
	return
}
func loadEnvConfig() Config {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	return Config{
		Host:  os.Getenv("HOST"),
		Port:  port,
		DBUrl: os.Getenv("DB_URL"),
	}
}

func main() {
	config := loadEnvConfig()

	storage := &SqlStorage{ctx: context.Background()}
	err := storage.init(config.DBUrl)
	if err != nil {
		log.Fatal(err.Error())
	}
	inbox := &Inbox{storage: storage}

	mux := http.NewServeMux()
	mux.HandleFunc("/send", inbox.sendHandler)
	mux.HandleFunc("/read", inbox.readHandler)
	mux.HandleFunc("/set-read", inbox.setHandler)

	c := cors.New(cors.Options{
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowCredentials: true,
		AllowedHeaders:   []string{"Zenflows-Sign"},
	})

	handler := c.Handler(mux)
	host := fmt.Sprintf("%s:%d", config.Host, config.Port)
	fmt.Printf("Starting service on %s\n", host)
	log.Fatal(http.ListenAndServe(host, handler))
}
