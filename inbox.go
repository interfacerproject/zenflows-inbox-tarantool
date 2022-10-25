//go:build ignore

package main

import (
    "fmt"
    "log"
    "net/http"
    _ "embed"
    "encoding/json"
    "io"
    "github.com/go-redis/redis/v8"
    "context"
    "os"
    "strconv"
    b64 "encoding/base64"
)

type Config struct {
    Redis string
    Port  int
    Host  string
}

type Message struct {
	Sender       string  `json:"sender"`
	Receivers  []string  `json:"receiver"`
}

type Inbox struct {
    rds *redis.Client
    ctx context.Context
}

//go:embed zenflows-crypto/src/verify_graphql.zen
var VERIFY string;

func (inbox *Inbox) sendHandler(w http.ResponseWriter, r *http.Request) {
    // Setup json response
    w.Header().Set("Content-Type", "application/json")
    result := map[string]interface{} {
        "success": false,
    }
    defer json.NewEncoder(w).Encode(result)

    body, err := io.ReadAll(r.Body)
    if err != nil {
        result["success"] = false
        result["error"] = "Could not read the body of the request"
        return
    }
    zenroomData := ZenroomData {
        Gql: b64.StdEncoding.EncodeToString(body),
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
    zenroomData.requestPublicKey(message.Sender)
    err = zenroomData.isAuth()
    if err != nil {
        result["success"] = false
        result["error"] = err.Error()
        return
    }

    // For each receiver put the message in the inbox
    count := 0
    for i := 0; i < len(message.Receivers); i++ {
        err := inbox.rds.SAdd(inbox.ctx, message.Receivers[i], body).Err()
        log.Printf("Added message for: %s", message.Receivers[i])
        if err == nil {
            count = count + 1
        }
    }
    result["success"] = true
    result["count"] = count
    return
}

type ReadMessages struct {
    RequestId int    `json:"request_id"`
    Sender    string `json:"sender"`
}

func (inbox *Inbox) readHandler(w http.ResponseWriter, r *http.Request) {
    // Setup json response
    w.Header().Set("Content-Type", "application/json")
    result := map[string]interface{} {
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
    zenroomData := ZenroomData {
        Gql: b64.StdEncoding.EncodeToString(body),
        EdDSASignature: r.Header.Get("zenflows-sign"),
    }
    var readMessage ReadMessages
    err = json.Unmarshal(body, &readMessage)
    if err != nil {
        result["success"] = false
        result["error"] = err.Error()
        return
    }
    zenroomData.requestPublicKey(readMessage.Sender)
    err = zenroomData.isAuth()
    if err != nil {
        result["success"] = false
        result["error"] = err.Error()
        return
    }
    pipe := inbox.rds.Pipeline()

    // Read from redis and delete the messages
    rdsMessages := pipe.SMembers(inbox.ctx, readMessage.Sender)
    pipe.Del(inbox.ctx, readMessage.Sender)

    _, err = pipe.Exec(inbox.ctx)
    if err != nil {
        result["success"] = false
        result["error"] = err.Error()
        return
    }
    resultMessages := rdsMessages.Val()
    var messages []map[string]string
    for i := 0; i < len(resultMessages); i++ {
        var message map[string]string
        json.Unmarshal([]byte(resultMessages[i]), &message)
        messages = append(messages, message)
    }

    result["success"] = true
    result["request_id"] = readMessage.RequestId
    result["messages"] = messages
    return
}

func loadEnvConfig() Config {
    port, _ := strconv.Atoi(os.Getenv("PORT"))
    return Config {
        Host: os.Getenv("HOST"),
        Redis: os.Getenv("REDIS"),
        Port: port,
    }
}

func main() {
    config := loadEnvConfig()

    inbox := &Inbox{rds: redis.NewClient(&redis.Options{
        Addr: config.Redis,
        Password: "",
        DB: 0,
    }), ctx: context.Background()}
    
    http.HandleFunc("/send", inbox.sendHandler)
    http.HandleFunc("/read", inbox.readHandler)

    host := fmt.Sprintf("%s:%d", config.Host, config.Port)
    log.Fatal(http.ListenAndServe(host, nil))
}
