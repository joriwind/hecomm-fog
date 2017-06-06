package cilorawan

import (
	"context"
	"log"

	ns "github.com/joriwind/hecomm-fog/api/ns"
	"github.com/joriwind/hecomm-fog/iotInterface"
	"google.golang.org/grpc"
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
func NewNetworkClient(ctx context.Context, host string) (*NetworkClient, error) {
	//Does the fog use secured connection?
	var n NetworkClient
	var nsDialOptions []grpc.DialOption
	nsDialOptions = append(nsDialOptions, grpc.WithTransportCredentials(
		mustGetTransportCredentials(confCILorawanCert, confCILorawanKey, confCILorawanCA, true),
	))
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
