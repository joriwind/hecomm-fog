package cilorawan

//"github.com/brocaar/loraserver/api/as"

/* Functions to be implemented by server listener */
//TODO: implement functions

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	//"github.com/brocaar/lora-app-server/internal/common"

	"time"

	"github.com/joriwind/hecomm-api/hecomm"
	as "github.com/joriwind/hecomm-fog/api/as"
	"github.com/joriwind/hecomm-fog/iotInterface"
)

// ApplicationServerAPI implements the as.ApplicationServerServer interface.
type ApplicationServerAPI struct {
	ctx     context.Context
	comlink chan iotInterface.ComLinkMessage
	port    string
	options []grpc.ServerOption
}

// NewApplicationServerAPI returns a new ApplicationServerAPI.
func NewApplicationServerAPI(ctx context.Context, comlink chan iotInterface.ComLinkMessage) *ApplicationServerAPI {
	//Set static config
	var nsOpts []grpc.ServerOption
	nsOpts = append(nsOpts, grpc.Creds(mustGetTransportCredentials(confCILorawanCert, confCILorawanKey, confCILorawanCA, true)))

	return &ApplicationServerAPI{
		ctx:     ctx,
		comlink: comlink,
		port:    ":8001",
		options: nsOpts,
	}

}

//StartServer creates a new server
func (a *ApplicationServerAPI) StartServer() error {
	lis, err := net.Listen("tcp", a.port)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer(a.options...)
	//server := NewApplicationServerAPI(a.ctx, a.comlink)
	as.RegisterApplicationServerServer(grpcServer, a)
	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)
	log.Println("cilorawan: Start listening!")
	if err := grpcServer.Serve(lis); err != nil {
		return err
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
		InterfaceType: hecomm.CILorawan,
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

func mustGetTransportCredentials(tlsCert, tlsKey, caCert string, verifyClientCert bool) credentials.TransportCredentials {
	var caCertPool *x509.CertPool
	cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
	if err != nil {
		log.Fatalf("load key-pair error: %s\n", err)
	}

	if caCert != "" {
		rawCaCert, err := ioutil.ReadFile(caCert)
		if err != nil {
			log.Fatalf("load ca cert error: %s\n", err)
		}

		caCertPool = x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(rawCaCert)
	}

	if verifyClientCert {
		return credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
		})
	} else {
		return credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		})
	}
}
