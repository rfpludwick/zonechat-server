package main

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	server      *Server
	httpRequest *http.Request
	connection  *websocket.Conn
	egress      chan []byte
}

const (
	writeWait      = (10 * time.Second)
	pongWait       = (60 * time.Second)
	pingPeriod     = ((pongWait * 9) / 10)
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")

			for _, hostname := range config.AllowedHosts {
				if origin == "http://"+hostname {
					return true
				}
			}

			return false
		},
	}
)

func newClient(s *Server, conn *websocket.Conn, r *http.Request) *Client {
	client := &Client{
		server:      s,
		httpRequest: r,
		connection:  conn,
		egress:      make(chan []byte, 256),
	}

	s.joinClient(client)

	return client
}

func serveClientWebsocket(s *Server, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println("HTTP upgrader error:", err)

		return
	}

	c := newClient(s, conn, r)

	go c.readIngress()
	go c.sendEgress()
}

func (c *Client) readIngress() {
	defer func() {
		c.server.leaveClient(c)
		c.connection.Close()
	}()

	c.connection.SetReadLimit(maxMessageSize)
	c.connection.SetReadDeadline(time.Now().Add(pongWait))

	c.connection.SetPongHandler(func(string) error {
		c.connection.SetReadDeadline(time.Now().Add(pongWait))

		return nil
	})

	for {
		_, message, err := c.connection.ReadMessage() // todo handle bad data type

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("Error reading ingress for client from", c.httpRequest.RemoteAddr)
			}

			return
		}

		// todo log the message somewhere

		c.server.sendMessage(bytes.TrimSpace(bytes.Replace(message, newline, space, -1)))
	}
}

func (c *Client) sendEgress() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.connection.Close()
	}()

	for {
		select {
		case message, ok := <-c.egress:
			c.connection.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				if c.server.hasClient(c) {
					log.Println("Error getting channel egress for client from", c.httpRequest.RemoteAddr)
				}

				c.connection.WriteMessage(websocket.CloseMessage, []byte{})

				return
			}

			w, err := c.connection.NextWriter(websocket.TextMessage)

			if err != nil {
				log.Println("Error acquiring writer for client from", c.httpRequest.RemoteAddr)

				return
			}

			w.Write(message)

			// Get any more buffered messages after the one we just retrieved from the channel
			n := len(c.egress)

			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.egress)
			}

			if err := w.Close(); err != nil {
				log.Println("Error closing writer for client from", c.httpRequest.RemoteAddr)

				return
			}

		case <-ticker.C:
			c.connection.SetWriteDeadline(time.Now().Add(writeWait))

			if err := c.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Error pinging client from", c.httpRequest.RemoteAddr)

				return
			}
		}
	}
}

func (c *Client) closeEgress() {
	close(c.egress)
}
