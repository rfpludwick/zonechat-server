package main

import "log"

type Server struct {
	clients map[*Client]bool
	join    chan *Client
	leave   chan *Client
	message chan []byte
}

func newServer() *Server {
	return &Server{
		clients: make(map[*Client]bool),
		join:    make(chan *Client),
		leave:   make(chan *Client),
		message: make(chan []byte),
	}
}

func (s *Server) run() {
	for {
		select {
		case c := <-s.join:
			s.clients[c] = true

		case c := <-s.leave:
			if _, joined := s.clients[c]; joined {
				s.closeClient(c)
			}

		case message := <-s.message:
			for c := range s.clients {
				select {
				case c.egress <- message:
					// No-op
					// Note: I wish I could use a receiver function on the Client class to handle this, rather than directly accessing the channel from Server

				default:
					s.closeClient(c)
				}
			}
		}
	}
}

func (s *Server) joinClient(c *Client) {
	s.join <- c

	log.Println("Client joined from", c.httpRequest.RemoteAddr)
}

func (s *Server) hasClient(c *Client) bool {
	_, ok := s.clients[c]

	return ok
}

func (s *Server) leaveClient(c *Client) {
	s.leave <- c

	log.Println("Client left from", c.httpRequest.RemoteAddr)
}

func (s *Server) sendMessage(m []byte) {
	s.message <- m
}

func (s *Server) closeClient(c *Client) {
	delete(s.clients, c)

	c.closeEgress()

	log.Println("Closing client from", c.httpRequest.RemoteAddr)
}
