package cilorawan

import (
	"log"

	ns "github.com/joriwind/hecomm-fog/api/ns"
	"google.golang.org/grpc"
)

//ConnectToNetworkServer Create connection with LoRaWAN Network server
func ConnectToNetworkServer(host string, nsDialOptions []grpc.DialOption) (*grpc.ClientConn, ns.NetworkServerClient) {
	//Does the fog use secured connection?
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
	}
	//defer asConn.Close() //TODO: Do not forget to close connection!
	return nsConn, ns.NewNetworkServerClient(nsConn)
}
