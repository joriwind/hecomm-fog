package cisixlowpan

import "github.com/joriwind/hecomm-fog/iotInterface"

import "fmt"
import "log"
import "io"
import "github.com/joriwind/hecomm-interface-6lowpan"

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

	n, err := c.conn.Write(message.Data)
	if err != nil {
		return fmt.Errorf("cisixlowpan: did not send error: %v", err)
	}
	log.Printf("cisixlowpan: send %v bytes to %v", n, message.Destination)
	return nil
}

//Close the connection
func (c *Client) Close() {
	c.conn.Close()
}
