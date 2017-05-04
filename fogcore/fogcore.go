package fogcore

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"

	"github.com/joriwind/hecomm-fog/dbconnection"
	"github.com/joriwind/hecomm-fog/iotInterface"
	"github.com/joriwind/hecomm-fog/iotInterface/cilorawan"
	"github.com/joriwind/hecomm-fog/iotInterface/cisixlowpan"
	"google.golang.org/grpc"
)

//Fogcore Struct
type Fogcore struct {
	ctx             context.Context
	opt             Options
	controlChannel  chan controlMessage
	ciCommonChannel chan iotInterface.ComLinkMessage
	ciCollection    []*ci
}

//Options Defines possible options to pass along with Fogcore object
type Options struct {
	Hostname   string
	CertServer string
	KeyServer  string
}

type ci struct {
	platform *dbconnection.Platform
	channel  chan iotInterface.ComLinkMessage
	ctx      context.Context
	cancel   func()
}

type controlMessage struct {
	command  string
	platform *dbconnection.Platform
}

//NewFogcore Create new fogcore module
func NewFogcore(ctx context.Context, opt Options) *Fogcore {
	fogcore := Fogcore{ctx: ctx, opt: opt}

	switch {
	case opt.Hostname == "":
		fogcore.opt.Hostname = "0.0.0.0:8000"
	case opt.KeyServer == "":
		fogcore.opt.KeyServer = "certs/server.key"
	case opt.CertServer == "":
		fogcore.opt.CertServer = "certs/server.pem"
	}

	return &fogcore
}

//Start Start the fogcore module
func (f *Fogcore) Start() error {
	//Start management interface
	go f.listenOnTLS()

	//Startup already known platforms
	platforms, err := dbconnection.GetPlatforms()
	if err != nil {
		log.Fatalf("fogcore: something went wrong in retrieving interfaces: %v", err)
	}
	//Create access to the will be routines of iot interfaces
	f.ciCollection = make([]*ci, len(platforms))
	f.ciCommonChannel = make(chan iotInterface.ComLinkMessage, 20)
	//Create the communication to the iot interface thread

	//Startup already known interfaces
	for index, pl := range platforms {
		channel := make(chan iotInterface.ComLinkMessage, 5)
		ctx, cancel := context.WithCancel(f.ctx)
		f.ciCollection[index] = &ci{platform: pl, channel: channel, ctx: ctx, cancel: cancel}
		f.startInterface(f.ciCollection[index])
	}

	for {
		select {
		case cm := <-f.controlChannel:
			if err := f.executeCommand(&cm); err != nil {
				log.Fatalf("Error in executeCommand! controlMessage: %v\n", cm)
			}
		case clm := <-f.ciCommonChannel:
			if err := f.handleCIMessage(&clm); err != nil {
				log.Fatalf("Error in handleCIMessage! message: %v\n", clm)
			}
		case <-f.ctx.Done():
			return nil
		}
	}
}

func (f *Fogcore) listenOnTLS() error {
	cert, err := tls.LoadX509KeyPair(f.opt.CertServer, f.opt.KeyServer)
	if err != nil {
		log.Fatalf("fogcore: tls error: loadkeys: %s", err)
		return err
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}}
	config.Rand = rand.Reader
	listener, err := tls.Listen("tcp", f.opt.Hostname, &config)
	if err != nil {
		log.Fatalf("fogcore: tls error: listen: %s", err)
		return err
	}
	defer listener.Close()

	//Listen for new tls connections
	newConns := make(chan net.Conn)
	go func() {
		log.Printf("fogcore: listening on TLS socket: %v", f.opt.Hostname)
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("fogcore: TLS accept error: %s", err)
				newConns <- nil
				return
			}
			newConns <- conn

		}
	}()

	//Wait for new tls connections or quit
	for {
		select {
		case conn := <-newConns:
			if conn == nil {
				return errors.New("fogcore: fail on TLS accept")
			}
			defer conn.Close()

			log.Printf("fogcore: accepted TLS connection from %s", conn.RemoteAddr())
			tlscon, ok := conn.(*tls.Conn)
			if ok {
				log.Print("ok=true")
				state := tlscon.ConnectionState()
				for _, v := range state.PeerCertificates {
					log.Print(x509.MarshalPKIXPublicKey(v.PublicKey))
				}
			}
			go handleTLSClient(conn)
		case <-f.ctx.Done():
			return nil
		}
	}

}

func handleTLSClient(conn net.Conn) {

}

//executeCommand Handle the control messages
func (f *Fogcore) executeCommand(message *controlMessage) error {
	switch message.command {
	case "push": //Start new platform
		channel := make(chan iotInterface.ComLinkMessage, 5)
		ctx, cancel := context.WithCancel(f.ctx)
		//Add new interface to end of collection
		f.ciCollection = append(f.ciCollection, &ci{platform: message.platform, channel: channel, ctx: ctx, cancel: cancel})
		//Startup last added interface
		f.startInterface(f.ciCollection[len(f.ciCollection)-1])

	case "pull": //Stop a platform
		for index, intface := range f.ciCollection {
			if intface.platform.ID == message.platform.ID {
				//Delete while preserving order
				//append the slice part before the element with all the elements after the specific element
				f.ciCollection = append(f.ciCollection[:index], f.ciCollection[index+1:]...)

			}

		}
	default:
		log.Fatalf("fogcore: unkown control message: %v\n", *message)

	}

	return nil
}

//startInterface Start listening on new interface
func (f *Fogcore) startInterface(iot *ci) error {
	//Starup the listening routines for all platforms in the database

	//Depending on type, create iot interface routine
	switch iot.platform.CIType {
	case iotInterface.Lorawan:

		//Convert general mapping to cilorawan Server Options
		var args []grpc.ServerOption
		cilorawan.ConvertArgsToUplinkOptions(iot.platform.CIArgs["uplink"], &args)

		//args := (grpc.ServerOption)pl.CIArgs
		lorawanapi := cilorawan.NewApplicationServerAPI(iot.ctx, iot.channel, args...)

		//Start the cilorawan
		go func() {
			if err := lorawanapi.StartServer(); err != nil {
				log.Fatalf("Something went wrong in interface: %v; error: %v", iot, err)
			}
		}()
		//Tunnel the communication to common channel -- easy access in main loop
		go func() {
			for {
				f.ciCommonChannel <- <-iot.channel
			}
		}()

	case iotInterface.Sixlowpan:
		//Create the communication to the iot interface thread

		var args cisixlowpan.ServerOptions
		cisixlowpan.ConvertArgsToUplinkOptions(iot.platform.CIArgs["uplink"], &args)

		sixlowpanServer := cisixlowpan.NewServer(iot.ctx, iot.channel, args)

		//Start the cilorawan
		go func() {
			if err := sixlowpanServer.Start(); err != nil {
				log.Fatalf("Something went wrong in interface: %v; error: %v", iot, err)
			}
		}()
		//Tunnel the communication to common channel -- easy access in main loop
		go func() {
			for {
				f.ciCommonChannel <- <-iot.channel
			}
		}()

	default:
		log.Fatalf("Unkown interface requested! %v", iot)
	}

	return nil
}

func (f *Fogcore) handleCIMessage(clm *iotInterface.ComLinkMessage) error {
	//Find destination node
	dstnode, err := dbconnection.GetDestination(clm)
	if err != nil {
		log.Fatalf("fogcore: Error in searching for destination node, message: %v", *clm)
		return err
	}
	platform, err := dbconnection.GetPlatform(dstnode.PlatformID)
	if err != nil {
		log.Fatalf("fogcore: Error in searching for platform of destination node, dstnode: %v", dstnode)
		return err

	}

	//Send to destination node
	switch platform.CIType {
	case iotInterface.Lorawan:
		var opt []grpc.DialOption
		//Get interface options for downlink
		err := cilorawan.ConvertArgsToDownlinkOption(platform.CIArgs["downlink"], &opt)
		if err != nil {
			log.Fatalf("focore: cilorawan: downlink conversion failed: options: %v/n", platform.CIArgs["downlink"])
			return err
		}

		//Create client, to send the message
		client, err := cilorawan.NewNetworkClient(context.Background(), platform.CIArgs["downlinkAddress"].(string), opt...)
		if err != nil {
			log.Fatalf("fogcore: cilorawan: creation of newnetworkclient failed! address: %v options: %v\n", platform.CIArgs["downlinkAddress"].(string), opt)
			return err
		}
		defer client.Close()
		//Send data with created client
		err = client.SendData(*clm)
		if err != nil {
			log.Fatalf("fogcore: cilorawan: unable to send message: %v", *clm)
			return err
		}

	case iotInterface.Sixlowpan:
		client, err := cisixlowpan.NewClient("udp6", dstnode.DevID)
		if err != nil {
			log.Fatalf("fogcore: cisixlowpan: unable to create client, destination: %v\n", dstnode.DevID)
			return err
		}
		defer client.Close()

		err = client.SendData(*clm)
		if err != nil {
			log.Fatalf("fogcore: cisixlowpan: unable to send message: %v, error: %v\n", *clm, err)
			return err
		}

	default:
		log.Fatalln("fogcore: Unkown interface of destination node")
	}
	return nil
}
