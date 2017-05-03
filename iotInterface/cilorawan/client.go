package cilorawan

import (
	"context"
	"encoding/json"
	"log"

	"fmt"

	"time"

	ns "github.com/joriwind/hecomm-fog/api/ns"
	"github.com/joriwind/hecomm-fog/iotInterface"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

//NetworkClient Object for interfacing LoRaWAN Network server client
type NetworkClient struct {
	ctx                 context.Context
	host                string
	nsDialOptions       []grpc.DialOption
	nsConn              *grpc.ClientConn
	networkServerClient ns.NetworkServerClient
}

//NewNetworkClient Create connection with LoRaWAN Network server
func NewNetworkClient(ctx context.Context, host string, nsDialOptions []grpc.DialOption) (*NetworkClient, error) {
	//Does the fog use secured connection?
	var n NetworkClient
	//var asDialOptions []grpc.DialOption
	/*if c.String("as-tls-cert") != "" && c.String("as-tls-key") != "" {
		asDialOptions = append(asDialOptions, grpc.WithTransportCredentials(
			mustGetTransportCredentials(c.String("as-tls-cert"), c.String("as-tls-key"), c.String("as-ca-cert"), false),
		))
	} else {*/
	nsDialOptions = append(nsDialOptions, grpc.WithInsecure())
	//}
	//host := "192.168.1.1:8000"
	nsConn, err := grpc.Dial(host, nsDialOptions...) //TODO: when close connection?
	if err != nil {
		log.Fatalf("application-server (FOG) dial error: %s", err)
		return &n, err
	}
	//defer asConn.Close() //TODO: Do not forget to close connection!
	networkServerClient := ns.NewNetworkServerClient(nsConn)
	n = NetworkClient{
		ctx:                 ctx,
		host:                host,
		nsDialOptions:       nsDialOptions,
		nsConn:              nsConn,
		networkServerClient: networkServerClient,
	}
	return &n, nil
}

//ConvertArgsToDownlinkOption Converts the general mapping to usable dialoptions
func ConvertArgsToDownlinkOption(args interface{}, opt *[]grpc.DialOption) error {
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

	//Loop over all the avalable options
	for index, value := range input {
		switch index {

		case "WithThransportCredentials":
			bytes, err := json.Marshal(value)
			if err != nil {
				return err
			}
			var option credentials.TransportCredentials
			err = json.Unmarshal(bytes, &option)
			if err != nil {
				return err
			}
			*opt = append(*opt, grpc.WithTransportCredentials(option))

		case "WithTimeout":
			bytes, err := json.Marshal(value)
			if err != nil {
				return err
			}
			var option time.Duration
			err = json.Unmarshal(bytes, &option)
			if err != nil {
				return err
			}
			*opt = append(*opt, grpc.WithTimeout(option))

		case "WithInsecure":
			*opt = append(*opt, grpc.WithInsecure())

		default:
			return fmt.Errorf("cilorawan: ConvertArgsToDownlinkOption: unkown option: %v: %v", index, value)
		}

	}
	return nil
}

//SendData Send data from fogCore to LoRaWAN Network Server
func (n *NetworkClient) SendData(message iotInterface.ComLinkMessage) error {
	pushDataDownReq := &ns.PushDataDownRequest{
		DevEUI:    message.Destination,
		Confirmed: true,
		FCnt:      0,
		FPort:     255,
		Data:      message.Data,
	}

	//Find the right FCnt
	nodeSessionRequest := &ns.GetNodeSessionRequest{
		DevEUI: message.Destination,
	}
	if nodeSessionResponse, err := n.networkServerClient.GetNodeSession(n.ctx, nodeSessionRequest, nil); err != nil {
		log.Printf("LoRaWAN interface: GetNodeSession did not work: %v", err)
	} else {
		pushDataDownReq.FCnt = nodeSessionResponse.FCntDown
	}

	//Send packet down to Network server
	if _, err := n.networkServerClient.PushDataDown(n.ctx, pushDataDownReq, nil); err != nil {
		return err
	}

	return nil
}

//Close Close the connection!
func (n *NetworkClient) Close() {
	n.nsConn.Close()
}
