package main

import (
	"context"
	_ "embed"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

type Config struct {
	port   int
	host   string
	ttHost string
	ttUser string
	ttPass string
	zfUrl  string
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
	countUnread(string) (int, error)
	delete(string, int) error
}

type Inbox struct {
	storage       Storage
	zfUrl         string
	zenflowsAgent ZenflowsAgent
}

//go:embed zenflows-crypto/src/verify_graphql.zen
var VERIFY string

func (inbox *Inbox) sendHandler(c *gin.Context) {
	// Setup json response
	result := map[string]interface{}{
		"success": false,
	}
	defer c.JSON(http.StatusOK, result)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		result["error"] = "Could not read the body of the request"
		return
	}
	zenroomData := ZenroomData{
		Gql:            b64.StdEncoding.EncodeToString(body),
		EdDSASignature: c.Request.Header.Get("zenflows-sign"),
	}

	// Read a message object, I need the receivers
	var message Message
	err = json.Unmarshal(body, &message)
	if err != nil {
		result["error"] = err.Error()
		return
	}

	if len(message.Receivers) == 0 {
		result["error"] = "No receivers"
		return
	}

	if len(message.Content) == 0 {
		result["error"] = "Empty content"
		return
	}
	err = zenroomData.requestPublicKey(inbox.zfUrl, message.Sender)
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = zenroomData.isAuth()
	if err != nil {
		result["error"] = err.Error()
		return
	}

	// For each receiver put the message in the inbox
	count, err := inbox.storage.send(message)
	if err != nil {
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

func (inbox *Inbox) readHandler(c *gin.Context) {
	// Setup json response
	result := map[string]interface{}{
		"success": false,
	}
	defer c.JSON(http.StatusOK, result)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		result["error"] = err.Error()
		return
	}

	// Verify signature request
	zenroomData := ZenroomData{
		Gql:            b64.StdEncoding.EncodeToString(body),
		EdDSASignature: c.Request.Header.Get("zenflows-sign"),
	}
	var readMessage ReadMessages
	err = json.Unmarshal(body, &readMessage)
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = zenroomData.requestPublicKey(inbox.zfUrl, readMessage.Receiver)
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = zenroomData.isAuth()
	if err != nil {
		result["error"] = err.Error()
		return
	}
	messages, err := inbox.storage.read(readMessage.Receiver, readMessage.OnlyUnread)
	if err != nil {
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

func (inbox *Inbox) setHandler(c *gin.Context) {
	// Setup json response
	result := map[string]interface{}{
		"success": false,
	}
	defer c.JSON(http.StatusOK, result)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		result["error"] = err.Error()
		return
	}

	// Verify signature request
	zenroomData := ZenroomData{
		Gql:            b64.StdEncoding.EncodeToString(body),
		EdDSASignature: c.Request.Header.Get("zenflows-sign"),
	}
	var setMessage SetMessage
	err = json.Unmarshal(body, &setMessage)
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = zenroomData.requestPublicKey(inbox.zfUrl, setMessage.Receiver)
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = zenroomData.isAuth()
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = inbox.storage.set(setMessage.Receiver, setMessage.MessageId, setMessage.Read)
	if err != nil {
		result["error"] = err.Error()
		return
	}

	result["success"] = true
	return
}

type CountMessages struct {
	Receiver string `json:"receiver"`
}

func (inbox *Inbox) countHandler(c *gin.Context) {
	// Setup json response
	result := map[string]interface{}{
		"success": false,
	}
	defer c.JSON(http.StatusOK, result)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		result["error"] = err.Error()
		return
	}

	// Verify signature request
	zenroomData := ZenroomData{
		Gql:            b64.StdEncoding.EncodeToString(body),
		EdDSASignature: c.Request.Header.Get("zenflows-sign"),
	}
	var countMessages CountMessages
	err = json.Unmarshal(body, &countMessages)
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = zenroomData.requestPublicKey(inbox.zfUrl, countMessages.Receiver)
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = zenroomData.isAuth()
	if err != nil {
		result["error"] = err.Error()
		return
	}
	count, err := inbox.storage.countUnread(countMessages.Receiver)
	if err != nil {
		result["error"] = err.Error()
		return
	}

	result["success"] = true
	result["count"] = count
	return
}

type DeleteMessage struct {
	MessageId int    `json:"message_id"`
	Receiver  string `json:"receiver"`
}

func (inbox *Inbox) deleteHandler(c *gin.Context) {
	// Setup json response
	result := map[string]interface{}{
		"success": false,
	}
	defer c.JSON(http.StatusOK, result)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		result["error"] = err.Error()
		return
	}

	// Verify signature request
	zenroomData := ZenroomData{
		Gql:            b64.StdEncoding.EncodeToString(body),
		EdDSASignature: c.Request.Header.Get("zenflows-sign"),
	}
	var deleteMessage DeleteMessage
	err = json.Unmarshal(body, &deleteMessage)
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = zenroomData.requestPublicKey(inbox.zfUrl, deleteMessage.Receiver)
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = zenroomData.isAuth()
	if err != nil {
		result["error"] = err.Error()
		return
	}
	err = inbox.storage.delete(deleteMessage.Receiver, deleteMessage.MessageId)
	if err != nil {
		result["error"] = err.Error()
		return
	}

	result["success"] = true
	return
}

func (inbox *Inbox) profileHandler(c *gin.Context) {
	id := c.Param("id")

	baseUrl := fmt.Sprintf("%s/%s", os.Getenv("BASE_URL"), id)
	zfPerson, err := inbox.zenflowsAgent.GetPerson(id)
	if err != nil {
		panic(err)
	}

	m := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       baseUrl,
		"name":     zfPerson.Name,
		"inbox":    baseUrl + "/inbox",
		"outbox":   baseUrl + "/outbox",
		"type":     "Person",
		"summary":  zfPerson.Note,
	}
	var person vocab.ActivityStreamsPerson
	resolver, _ := streams.NewJSONResolver(func(c context.Context, p vocab.ActivityStreamsPerson) error {
		// Store the person in the enclosing scope, for later.
		person = p
		return nil
	}, func(c context.Context, note vocab.ActivityStreamsNote) error {
		// We can treat the type differently.
		fmt.Println(note)
		return nil
	})
	ctx := context.Background()
	resolver.Resolve(ctx, m)
	var jsonmap map[string]interface{}
	jsonmap, _ = streams.Serialize(person) // WARNING: Do not call the Serialize() method on person
	c.JSON(http.StatusOK, jsonmap)

}
func loadEnvConfig() Config {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	return Config{
		host:   os.Getenv("HOST"),
		port:   port,
		ttHost: os.Getenv("TT_HOST"),
		ttUser: os.Getenv("TT_USER"),
		ttPass: os.Getenv("TT_PASS"),
		zfUrl:  fmt.Sprintf("%s/api", os.Getenv("ZENFLOWS_URL")),
	}
}

func main() {
	config := loadEnvConfig()
	log.Printf("Using backend %s\n", config.zfUrl)

	za := ZenflowsAgent{
		Sk:          os.Getenv("ZENFLOWS_SK"),
		ZenflowsUrl: config.zfUrl,
	}

	storage := &TTStorage{}
	err := storage.init(config.ttHost, config.ttUser, config.ttPass)
	if err != nil {
		log.Fatal(err.Error())
	}
	inbox := &Inbox{
		storage:       storage,
		zfUrl:         config.zfUrl,
		zenflowsAgent: za,
	}

	r := gin.Default()
	r.Use(cors.Default())

	r.POST("/send", inbox.sendHandler)
	r.POST("/read", inbox.readHandler)
	r.POST("/set-read", inbox.setHandler)
	r.POST("/count-unread", inbox.countHandler)
	r.POST("/delete", inbox.deleteHandler)

	r.GET("/social/:id", inbox.profileHandler)

	host := fmt.Sprintf("%s:%d", config.host, config.port)
	log.Printf("Starting service on %s\n", host)
	r.Run()
}
