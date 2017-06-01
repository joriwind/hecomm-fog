package cisixlowpan

import (
	"context"
	"encoding/json"
	"fmt"

	"net"

	"log"

	"time"

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
func NewServer(ctx context.Context, comlink chan iotInterface.ComLinkMessage, opt ServerOptions) *Server {
	var server Server
	server.ctx = ctx
	server.comlink = comlink

	if opt.host != "" {
		server.host = opt.host
	} else {
		server.host = "[::1]:5683"
	}

	server.options = opt
	return &server
}

//Start Create socket and start listening
func (s *Server) Start() error {
	address, err := net.ResolveUDPAddr("udp6", s.host)
	if err != nil {
		log.Fatalf("cisixlowpan: unable to resolve UDP address: err: %v\n", err)
		return err
	}

	ln, err := net.ListenUDP("upd6", address)
	if err != nil {
		log.Fatalf("cisixlowpan: unable to listen on address: %v, error: %v", address, err)
		return err
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

//ConvertArgsToUplinkOptions Convert the stored general interface options from database into usable options for sixlowpan server
func ConvertArgsToUplinkOptions(args interface{}, opt *ServerOptions) error {
	//Convert to usable format
	argsBytes, err := json.Marshal(args)
	if err != nil {
		return err
	}
	var input map[string]interface{}
	err = json.Unmarshal(argsBytes, input)
	if err != nil {
		return err
	}

	//Loop over all available options
	for index, value := range input {
		switch index {

		default:
			return fmt.Errorf("cilorawan: ConvertARgsToOptions: unkown ServerOption: %v: %v", index, value)
		}
	}
	return nil
}
