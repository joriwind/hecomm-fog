package cisixlowpan

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/joriwind/hecomm-fog/iotInterface"
	"github.com/joriwind/hecomm-interface-6lowpan"
	"golang.org/x/net/ipv6"
)

//Client Link with sixlowpan destination
type Client struct {
	config sixlowpan.Config
	conn   io.WriteCloser
}

//NewClient Create connection with destination
func NewClient(config sixlowpan.Config) (*Client, error) {
	var err error

	client := Client{
		config: config,
		conn:   nil,
	}
	if config.PortName == "" {
		client.config = sixlowpan.Config{
			DebugLevel: sixlowpan.DebugAll,
			PortName:   "/dev/ttyUSB0",
		}
	} else {
		client.config = config
	}

	client.conn, err = sixlowpan.Open(config)
	if err != nil {
		return nil, err
	}

	return &client, nil
}

//SendData Send message to destination
func (c *Client) SendData(message iotInterface.ComLinkMessage) error {
	if c.conn == nil {
		return fmt.Errorf("cisixlowpan: connection not available: config: %v", c.config)
	}

	log.Println("Compiling packet for 6lowpan...")
	buf, err := compilePacket(message)
	if err != nil {
		return err
	}
	log.Printf("Sending: %x\n", buf)

	n, err := c.conn.Write(buf)
	if err != nil {
		return fmt.Errorf("cisixlowpan: did not send error: %v", err)
	}
	log.Printf("cisixlowpan: send %v bytes to %x", n, message.Destination)
	return nil
}

//Close the connection
func (c *Client) Close() {
	c.conn.Close()
}

func compilePacket(message iotInterface.ComLinkMessage) ([]byte, error) {
	//buf := make([]byte, ipv6.HeaderLen+sixlowpan.UdpHeaderLen+len(message.Data))

	iph := ipv6.Header{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		PayloadLen:   sixlowpan.UdpHeaderLen + len(message.Data),
		NextHeader:   17,
		HopLimit:     255,
		Src:          net.ParseIP("aaaa::c30c:0:0:5"), //TODO: variable source IP, depending on IP set for udp-slip
		Dst:          net.ParseIP(string(message.Destination[:]))}

	udph := sixlowpan.UDPHeader{
		DstPort: 5683,
		Length:  uint16(sixlowpan.UdpHeaderLen + len(message.Data)),
		Payload: message.Data,
		SrcPort: 5683,
		Chksum:  0,
	}

	err := udph.CalcChecksum(iph)
	if err != nil {
		return nil, err
	}

	ippayload, err := udph.Marschal()
	if err != nil {
		return nil, err
	}

	b, err := sixlowpan.Marschal(iph, ippayload)
	if err != nil {
		return nil, err
	}

	return b, nil
}
