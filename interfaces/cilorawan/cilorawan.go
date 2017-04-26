package cilorawan

import (
	"context"
	"log"
	"net"

	pb "github.com/joriwind/hecomm-fog/api/as"
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
