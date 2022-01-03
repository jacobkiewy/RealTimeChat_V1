package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	gubrak "github.com/novalagung/gubrak/v2"
)

var connections = make([]*WebSocketConnection, 0)

const MESSAGE_NEW_USER = "New User"
const MESSAGE_CHAT = "Chat"
const MESSAGE_LEAVE = "Leave"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  0,
	WriteBufferSize: 0, //1024
}

type SocketPayload struct {
	Message string
}

type SocketResponse struct {
	From    string
	Type    string
	Message string
}

type WebSocketConnection struct {
	*websocket.Conn
	Username string
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		content, err := ioutil.ReadFile("index.html")
		if err != nil {
			http.Error(w, "Dosya Açılamadı!", http.StatusInternalServerError)
		}
		fmt.Fprintf(w, "%s", content)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		gorillaConn, err := upgrader.Upgrade(w, r, w.Header())
		if err != nil {
			http.Error(w, "Websocket Bağlantısı Açılamadı", http.StatusInternalServerError)
		}
		user := r.URL.Query().Get("user")

		currentConn := WebSocketConnection{Conn: gorillaConn, Username: user}

		connections = append(connections, &currentConn)

		go handleIO(&currentConn, connections)
	})
	fmt.Println(":7000 working...")
	http.ListenAndServe(":7000", nil)
}

func handleIO(currentConn *WebSocketConnection, connections []*WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	sendMessage(currentConn, MESSAGE_NEW_USER, "")
	for {
		payload := SocketPayload{}
		err := currentConn.ReadJSON(&payload)
		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") {
				sendMessage(currentConn, MESSAGE_LEAVE, "")
				ejectConnections(currentConn)
				return
			}
			log.Println("ERROR", err.Error())
			continue
		}
		sendMessage(currentConn, MESSAGE_CHAT, payload.Message)
	}
}

func ejectConnections(currentConn *WebSocketConnection) {
	filtered := gubrak.From(connections).Reject(func(each *WebSocketConnection) bool {
		return each == currentConn
	}).Result()
	connections = filtered.([]*WebSocketConnection)
}

func sendMessage(currentConn *WebSocketConnection, kind, message string) {
	for _, eachConn := range connections {
		if eachConn == currentConn {
			continue
		}
		eachConn.WriteJSON(SocketResponse{
			From:    currentConn.Username,
			Type:    kind,
			Message: message,
		})
	}
}
