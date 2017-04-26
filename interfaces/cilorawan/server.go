package cilorawan

//"github.com/brocaar/loraserver/api/as"

/* Functions to be implemented by server listener */
//TODO: implement functions

import (
	"log"
	"net"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"

	//"github.com/brocaar/lora-app-server/internal/common"
	"github.com/brocaar/lorawan"
	as "github.com/joriwind/hecomm-fog/api/as"
	"github.com/joriwind/hecomm-fog/interfaces"
)

// ApplicationServerAPI implements the as.ApplicationServerServer interface.
type ApplicationServerAPI struct {
	ctx     context.Context
	comlink chan interfaces.ComLinkMessage
	port    string
}

// NewApplicationServerAPI returns a new ApplicationServerAPI.
func NewApplicationServerAPI(ctx context.Context, comlink chan interfaces.ComLinkMessage) *ApplicationServerAPI {
	return &ApplicationServerAPI{
		ctx:     ctx,
		comlink: comlink,
		port:    ":8000",
	}

}

//StartServer creates a new server
func (a *ApplicationServerAPI) StartServer() error {
	lis, err := net.Listen("tcp", a.port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Println("Created socket!")
	grpcServer := grpc.NewServer()
	server := NewApplicationServerAPI(a.ctx, a.comlink)
	as.RegisterApplicationServerServer(grpcServer, server)
	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)
	log.Println("Ready to listen!")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	return err
}

// JoinRequest handles a join-request.
func (a *ApplicationServerAPI) JoinRequest(ctx context.Context, req *as.JoinRequestRequest) (*as.JoinRequestResponse, error) {
	return nil, nil
}

// HandleDataUp handles incoming (uplink) data.
func (a *ApplicationServerAPI) HandleDataUp(ctx context.Context, req *as.HandleDataUpRequest) (*as.HandleDataUpResponse, error) {
	if len(req.RxInfo) == 0 {
		return nil, grpc.Errorf(codes.InvalidArgument, "RxInfo must have length > 0")
	}

	return nil, nil

}

// GetDataDown returns the first payload from the datadown queue.
func (a *ApplicationServerAPI) GetDataDown(ctx context.Context, req *as.GetDataDownRequest) (*as.GetDataDownResponse, error) {
	var devEUI lorawan.EUI64
	copy(devEUI[:], req.DevEUI)

	return nil, nil

}

// HandleDataDownACK handles an ack on a downlink transmission.
func (a *ApplicationServerAPI) HandleDataDownACK(ctx context.Context, req *as.HandleDataDownACKRequest) (*as.HandleDataDownACKResponse, error) {
	var devEUI lorawan.EUI64
	copy(devEUI[:], req.DevEUI)

	return nil, nil

}

// HandleError handles an incoming error.
func (a *ApplicationServerAPI) HandleError(ctx context.Context, req *as.HandleErrorRequest) (*as.HandleErrorResponse, error) {
	var devEUI lorawan.EUI64
	copy(devEUI[:], req.DevEUI)

	return nil, nil
}
