package main

import (
	"encoding/json"
	"fmt"
	mathrand "math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/sparkscience/go-wskeyid"
	"github.com/sparkscience/go-wskeyid/messages/clientmessage"
	"github.com/sparkscience/go-wskeyid/messages/servermessages"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func randInt(max int) int {
	return int(mathrand.Float32() * float32(max))
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.ReadFile("./index.html")
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Failed to read HTML file"))
			return
		}
		w.WriteHeader(200)
		w.Write(file)
	})

	router.HandleFunc("/path", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("New connection from client")
		defer fmt.Println("Connection closed")

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		{
			err := wskeyid.HandleAuthConnection(r, c)
			if err != nil {
				return
			}
		}

		fmt.Println("Connected")

		go func() {
			for {
				c.WriteJSON(map[string]interface{}{"type": "COOL"})
				<-time.After(time.Second * time.Duration(randInt(10)))
			}
		}()

		clientId := strings.TrimSpace(r.URL.Query().Get("client_id"))

		fmt.Printf("Connected to client with ID %s\n", clientId)

		go func() {
			err := c.WriteJSON(servermessages.Message{Type: "TEXT_MESSAGE", Data: "Cool"})
			fmt.Println("Sending message")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Got error %s", err.Error())
				return
			}
			<-time.After(time.Second * time.Duration(randInt(10)))
		}()

		for t, message, err := c.ReadMessage(); err == nil; {
			fmt.Printf("Got message")
			var m clientmessage.Message
			if t != websocket.BinaryMessage && t != websocket.TextMessage {
				continue
			}
			json.Unmarshal(message, &m)
			if m.Type != "RESPONSE" {
				fmt.Printf("Got message of type %s from %s\n", m.Type, clientId)
				continue
			}
			var str string
			err := m.UnmarshalData(&str)
			if err != nil {
				fmt.Printf("Failed to get message body")
			} else {
				fmt.Printf("Got message from client %s: %s\n", clientId, str)
			}
		}

		fmt.Println("Message stream ended between client and server. Closing connection")
	})

	fmt.Println("Server listening on port 8001")
	panic(http.ListenAndServe(":8001", router))
}
