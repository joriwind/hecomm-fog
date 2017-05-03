package cilorawan

//"github.com/brocaar/loraserver/api/as"

/* Functions to be implemented by server listener */
//TODO: implement functions

import (
	"log"
	"net"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	//"github.com/brocaar/lora-app-server/internal/common"

	"time"

	"encoding/json"

	"fmt"

	as "github.com/joriwind/hecomm-fog/api/as"
	"github.com/joriwind/hecomm-fog/iotInterface"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// ApplicationServerAPI implements the as.ApplicationServerServer interface.
type ApplicationServerAPI struct {
	ctx     context.Context
	comlink chan iotInterface.ComLinkMessage
	port    string
	options []grpc.ServerOption
}

// NewApplicationServerAPI returns a new ApplicationServerAPI.
func NewApplicationServerAPI(ctx context.Context, comlink chan iotInterface.ComLinkMessage, opt ...grpc.ServerOption) *ApplicationServerAPI {
	return &ApplicationServerAPI{
		ctx:     ctx,
		comlink: comlink,
		port:    ":8000",
		options: opt,
	}

}

//ConvertArgsToUplinkOptions Convert the stored general interface options from database into usable options for lorawan server
func ConvertArgsToUplinkOptions(args interface{}, opt *[]grpc.ServerOption) error {
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

	//Loop over all available options
	for index, value := range input {
		switch index {
		case "Creds":
			bytes, err := json.Marshal(value)
			if err != nil {
				return err
			}
			var option credentials.TransportCredentials
			err = json.Unmarshal(bytes, &option)
			if err != nil {
				return err
			}
			*opt = append(*opt, grpc.Creds(option))

		case "KeepaliveParams":
			bytes, err := json.Marshal(value)
			if err != nil {
				return err
			}
			var option keepalive.ServerParameters
			err = json.Unmarshal(bytes, &option)
			if err != nil {
				return err
			}
			*opt = append(*opt, grpc.KeepaliveParams(option))

		case "KeepaliveEnforcementPolicy":
			bytes, err := json.Marshal(value)
			if err != nil {
				return err
			}
			var option keepalive.EnforcementPolicy
			err = json.Unmarshal(bytes, &option)
			if err != nil {
				return err
			}
			*opt = append(*opt, grpc.KeepaliveEnforcementPolicy(option))

		default:
			return fmt.Errorf("cilorawan: ConvertARgsToOptions: unkown ServerOption: %v: %v", index, value)
		}
	}
	return nil
}

//StartServer creates a new server
func (a *ApplicationServerAPI) StartServer() error {
	lis, err := net.Listen("tcp", a.port)
	if err != nil {
		log.Fatalf("cilorawan: failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer(a.options...)
	//server := NewApplicationServerAPI(a.ctx, a.comlink)
	as.RegisterApplicationServerServer(grpcServer, a)
	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)
	log.Println("cilorawan: Start listening!")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("cilorawan: failed to serve: %v", err)
	}

	return err
}

// JoinRequest handles a join-request.
func (a *ApplicationServerAPI) JoinRequest(ctx context.Context, req *as.JoinRequestRequest) (*as.JoinRequestResponse, error) {
	log.Println("cilorawan: received JoinRequest???")
	return &as.JoinRequestResponse{}, nil
}

// HandleDataUp handles incoming (uplink) data.
func (a *ApplicationServerAPI) HandleDataUp(ctx context.Context, req *as.HandleDataUpRequest) (*as.HandleDataUpResponse, error) {
	/*if len(req.RxInfo) == 0 {
		return nil, grpc.Errorf(codes.InvalidArgument, "RxInfo must have length > 0")
	}*/
	log.Printf("cilorawan: Received data from %v: %v", req.DevEUI, req.Data)

	message := iotInterface.ComLinkMessage{
		Data:          req.Data,
		Destination:   nil,
		InterfaceType: iotInterface.Lorawan,
		Origin:        req.DevEUI,
		TimeReceived:  time.Now(),
	}
	a.comlink <- message

	return &as.HandleDataUpResponse{}, nil

}

// GetDataDown returns the first payload from the datadown queue.
func (a *ApplicationServerAPI) GetDataDown(ctx context.Context, req *as.GetDataDownRequest) (*as.GetDataDownResponse, error) {
	/*var devEUI lorawan.EUI64
	copy(devEUI[:], req.DevEUI)*/

	return nil, nil

}

// HandleDataDownACK handles an ack on a downlink transmission.
func (a *ApplicationServerAPI) HandleDataDownACK(ctx context.Context, req *as.HandleDataDownACKRequest) (*as.HandleDataDownACKResponse, error) {
	/*var devEUI lorawan.EUI64
	copy(devEUI[:], req.DevEUI)*/

	return nil, nil

}

// HandleError handles an incoming error.
func (a *ApplicationServerAPI) HandleError(ctx context.Context, req *as.HandleErrorRequest) (*as.HandleErrorResponse, error) {
	/*var devEUI lorawan.EUI64
	copy(devEUI[:], req.DevEUI)*/
	log.Println("cilorawan: HandleError request???")

	return nil, nil
}
