package cisixlowpan

import (
	"context"

	"net"

	"log"

	"time"

	"github.com/joriwind/hecomm-fog/iotInterface"
)

//Server Object defining the Server
type Server struct {
	ctx     context.Context
	comlink chan iotInterface.ComLinkMessage
	port    string
	options ServerOptions
}

//ServerOptions Options for the cisixlowpan server
type ServerOptions struct {
}

//NewServer Setup the cisixlowpan server
func NewServer(ctx context.Context, comlink chan iotInterface.ComLinkMessage, opt ServerOptions) *Server {
	return &Server{
		ctx:     ctx,
		comlink: comlink,
		port:    ":5656",
		options: opt,
	}
}

//Start Create socket and start listening
func (s *Server) Start() {
	address, err := net.ResolveUDPAddr("udp6", s.port)
	if err != nil {
		log.Fatalf("cisixlowpan: unable to resolve UDP address: err: %v\n", err)
	}

	ln, err := net.ListenUDP("upd6", address)
	if err != nil {
		log.Fatalf("cisixlowpan: unable to listen on address: %v, error: %v", address, err)
	}
	defer ln.Close()

	buf := make([]byte, 1024)

	for {
		n, addr, err := ln.ReadFrom(buf)
		if err != nil {
			log.Fatalf("cisixlowpan: failed at reading UDP packet: from: %v, error: %v\n", addr, err)
		}
		//Send packet to fogcore
		s.handlePacket(buf[0:n], addr)

	}

}

//handlePacket Sends packet to fogcore
func (s *Server) handlePacket(buf []byte, addr net.Addr) {
	var message iotInterface.ComLinkMessage
	message = iotInterface.ComLinkMessage{
		Data:          buf,
		InterfaceType: iotInterface.Sixlowpan,
		Origin:        []byte(addr.String()),
		TimeReceived:  time.Now(),
		Destination:   nil,
	}

	s.comlink <- message
}
