package cisixlowpan

import "github.com/joriwind/hecomm-fog/iotInterface"
import "net"
import "fmt"
import "log"

//Client Link with sixlowpan destination
type Client struct {
	protocol string
	dstAddr  string
	conn     net.Conn
}

//NewClient Create connection with destination
func NewClient(protocol string, destinationAddress string) (*Client, error) {
	client := Client{
		protocol: protocol,
		dstAddr:  destinationAddress,
		conn:     nil,
	}

	conn, err := net.Dial(client.protocol, client.dstAddr)
	if err != nil {
		return &client, err
	}
	client.conn = conn

	return &client, nil
}

//SendData Send message to destination
func (c *Client) SendData(message iotInterface.ComLinkMessage) error {
	if c.conn == nil {
		return fmt.Errorf("cisixlowpan: connection not available: dstAddr: %v, protocol: %v", c.dstAddr, c.protocol)
	}

	n, err := c.conn.Write(message.Data)
	if err != nil {
		return fmt.Errorf("cisixlowpan: did not send to %v, error: %v", c.dstAddr, err)
	}
	log.Printf("cisixlowpan: send %v packet with %v bytes to %v", c.protocol, n, c.dstAddr)
	return nil
}

//Close the connection
func (c *Client) Close() {
	c.Close()
}
