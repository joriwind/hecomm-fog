package cisixlowpan

import (
	"context"

	"net"

	"log"

	"time"

	"github.com/joriwind/hecomm-api/hecomm"
	"github.com/joriwind/hecomm-fog/iotInterface"
)

//Server Object defining the Server
type Server struct {
	ctx     context.Context
	comlink chan iotInterface.ComLinkMessage
	host    string
	options ServerOptions
}

//ServerOptions Options for the cisixlowpan server
type ServerOptions struct {
	host string
}

//NewServer Setup the cisixlowpan server
func NewServer(ctx context.Context, comlink chan iotInterface.ComLinkMessage) *Server {
	var server Server
	server.ctx = ctx
	server.comlink = comlink

	server.host = confCISixlowpanAddress

	return &server
}

//Start Create socket and start listening
func (s *Server) Start() error {
	address, err := net.ResolveUDPAddr("udp6", s.host)
	if err != nil {
		return err
	}

	ln, err := net.ListenUDP("udp6", address)
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Printf("cisixlowpan: listening on %v\n", ln.LocalAddr())

	buf := make([]byte, 1024)

	for {
		n, addr, err := ln.ReadFrom(buf)
		if err != nil {
			return err
		}
		log.Printf("Received packet from %v\n", addr.String())
		//Send packet to fogcore
		s.handlePacket(buf[0:n], addr)

	}

}

//handlePacket Sends packet to fogcore
func (s *Server) handlePacket(buf []byte, addr net.Addr) {
	var message iotInterface.ComLinkMessage
	message = iotInterface.ComLinkMessage{
		Data:          buf,
		InterfaceType: hecomm.CISixlowpan,
		Origin:        []byte(addr.String()),
		TimeReceived:  time.Now(),
		Destination:   nil,
	}

	s.comlink <- message
}
