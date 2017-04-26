package cilorawan

import (
	"context"
	"log"
	"net"

	pb "github.com/joriwind/hecomm-fog/api/as"
	ns "github.com/joriwind/hecomm-fog/api/ns"
	"github.com/joriwind/hecomm-fog/interfaces"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	portGrpcServer = ":8000"
)

//Server Is used to implement application server
type server struct {
}

//StartServer creates a new server
func StartServer(ctx context.Context, comLink chan interfaces.ComLinkMessage) error {
	lis, err := net.Listen("tcp", portGrpcServer)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Println("Created socket!")
	grpcServer := grpc.NewServer()
	server := NewApplicationServerAPI(ctx, comLink)
	pb.RegisterApplicationServerServer(grpcServer, server)
	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)
	log.Println("Ready to listen!")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	return err
}

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
