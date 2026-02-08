package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ChatMessage struct {
	Type  string   `json:"type"`
	User  string   `json:"user"`
	Text  string   `json:"text"`
	Users []string `json:"users"`
}

type ChatClient struct {
	connection *websocket.Conn
	username   string
	clientID   string
}

var (
	activeClients = make(map[*ChatClient]bool)
	clientsLock   sync.Mutex

	wsUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func main() {
	router := gin.Default()

	router.GET("/ws", handleWebSocket)
	router.GET("/", serveIndexPage)
	router.Static("/static", "../frontend")

	log.Println("Server running on :8080")
	router.Run(":8080")
}

func serveIndexPage(c *gin.Context) {
	c.File("../frontend/index.html")
}

func handleWebSocket(c *gin.Context) {
	wsConn, _ := wsUpgrader.Upgrade(c.Writer, c.Request, nil)

	client := &ChatClient{
		connection: wsConn,
		clientID:   uuid.NewString(),
	}

	clientsLock.Lock()
	activeClients[client] = true
	clientsLock.Unlock()

	sendActiveUsers()

	defer func() {
		clientsLock.Lock()
		delete(activeClients, client)
		clientsLock.Unlock()

		sendActiveUsers()
		wsConn.Close()
	}()

	for {
		var incomingMsg ChatMessage
		if wsConn.ReadJSON(&incomingMsg) != nil {
			break
		}

		switch incomingMsg.Type {
		case "join":
			client.username = incomingMsg.User
			sendActiveUsers()

		case "message":
			broadcastMessage(incomingMsg)
		}
	}
}

func broadcastMessage(msg ChatMessage) {
	clientsLock.Lock()
	defer clientsLock.Unlock()

	for client := range activeClients {
		client.connection.WriteJSON(msg)
	}
}

func sendActiveUsers() {
	var usernames []string

	clientsLock.Lock()
	for client := range activeClients {
		if client.username != "" {
			usernames = append(usernames, client.username)
		}
	}
	clientsLock.Unlock()

	broadcastMessage(ChatMessage{
		Type:  "users",
		Users: usernames,
	})
}
