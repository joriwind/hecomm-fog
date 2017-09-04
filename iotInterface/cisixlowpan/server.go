package cisixlowpan

import (
	"context"
	"fmt"

	"golang.org/x/net/ipv6"

	"log"

	"time"

	"github.com/joriwind/hecomm-api/hecomm"
	"github.com/joriwind/hecomm-fog/iotInterface"
	"github.com/joriwind/hecomm-interface-6lowpan"
)

//Server Object defining the Server
type Server struct {
	ctx     context.Context
	comlink chan iotInterface.ComLinkMessage
	options sixlowpan.Config
}

//NewServer Setup the cisixlowpan server
func NewServer(ctx context.Context, comlink chan iotInterface.ComLinkMessage, config sixlowpan.Config) *Server {
	var server Server
	server.ctx = ctx
	server.comlink = comlink
	if config.PortName == "" {
		server.options = sixlowpan.Config{
			DebugLevel: sixlowpan.DebugAll,
			PortName:   "/dev/ttyUSB0",
		}
	} else {
		server.options = config
	}

	return &server
}

//Start Create socket and start listening
func (s *Server) Start() error {
	reader, err := sixlowpan.Open(s.options)
	if err != nil {
		return err
	}
	defer reader.Close()

	log.Printf("cisixlowpan: listening on %v\n", s.options.PortName)

	buf := make([]byte, 1024)

	for {
		//n, addr, err := ln.ReadFrom(buf)
		n, err := reader.Read(buf)
		if err != nil {
			return err
		}

		log.Printf("Received packet from SLIP\n")
		message, err := toComLinkMessage(buf[0:n])
		if err != nil {
			log.Printf("Could not translate slip packet: %v\n", err)

		} else {
			//Communicate to main thread
			log.Printf("Packet succesfully parsed, received from: %v", message.Origin)
			s.comlink <- message
		}

		//Check if ctx is expired
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}

	}

}

//toComLinkMessage parses ipv6 & udp header to create comlinkmessage
func toComLinkMessage(buf []byte) (m iotInterface.ComLinkMessage, err error) {
	if len(buf) < (ipv6.HeaderLen + sixlowpan.UdpHeaderLen) {
		return m, fmt.Errorf("Buf to small, could not fit ipv6 + udp header")
	}
	//Parsing the ip header to get source address
	iph, err := ipv6.ParseHeader(buf[:ipv6.HeaderLen])
	if err != nil {
		return m, err
	}

	//Unmarshalling UDP header to get to the payload
	udph, err := sixlowpan.UnmarshalUDP(buf[ipv6.HeaderLen:])
	if err != nil {
		return m, err
	}

	m = iotInterface.ComLinkMessage{
		Data:          udph.Payload,
		InterfaceType: hecomm.CISixlowpan,
		Origin:        []byte(iph.Src.String()),
		TimeReceived:  time.Now(),
		Destination:   nil, //Destination was fog, now should be something else
	}
	return m, err
}
