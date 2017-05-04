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
	ctx context.Context
	opt Options
}

type ci struct {
	ctx     context.Context
	ciType  string
	comLink chan []byte
}

//Options Defines possible options to pass along with Fogcore object
type Options struct {
	Hostname   string
	CertServer string
	KeyServer  string
}

type iotReference struct {
	platform *dbconnection.Platform
	channel  chan iotInterface.ComLinkMessage
	ctx      context.Context
	cancel   func()
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
	iotInterfaces := make([]*iotReference, len(platforms))
	iotChannel := make(chan iotInterface.ComLinkMessage, 20)

	//Starup the listening routines for all platforms in the database
	for index, pl := range platforms {
		//Depending on type, create iot interface routine
		switch pl.CIType {
		case iotInterface.Lorawan:
			//Create the communication to the iot interface thread
			channel := make(chan iotInterface.ComLinkMessage, 5)
			ctx, cancel := context.WithCancel(f.ctx)

			//Convert general mapping to cilorawan Server Options
			var args []grpc.ServerOption
			cilorawan.ConvertArgsToUplinkOptions(pl.CIArgs["uplink"], &args)

			//args := (grpc.ServerOption)pl.CIArgs
			lorawanapi := cilorawan.NewApplicationServerAPI(f.ctx, channel, args...)

			iotInterfaces[index] = &iotReference{platform: pl, channel: channel, ctx: ctx, cancel: cancel}
			//Start the cilorawan
			go func() {
				if err := lorawanapi.StartServer(); err != nil {
					log.Fatalf("Something went wrong in interface: %v; error: %v", iotInterfaces[index], err)
				}
			}()
			//Tunnel the communication to common channel -- easy access in main loop
			go func() {
				for {
					iotChannel <- <-channel
				}
			}()

		case iotInterface.Sixlowpan:
			//Create the communication to the iot interface thread
			channel := make(chan iotInterface.ComLinkMessage, 5)
			ctx, cancel := context.WithCancel(f.ctx)

			var args cisixlowpan.ServerOptions
			cisixlowpan.ConvertArgsToUplinkOptions(pl.CIArgs["uplink"], &args)

			sixlowpanServer := cisixlowpan.NewServer(ctx, channel, args)

			iotInterfaces[index] = &iotReference{platform: pl, channel: channel, ctx: ctx, cancel: cancel}
			//Start the cilorawan
			go func() {
				if err := sixlowpanServer.Start(); err != nil {
					log.Fatalf("Something went wrong in interface: %v; error: %v", iotInterfaces[index], err)
				}
			}()
			//Tunnel the communication to common channel -- easy access in main loop
			go func() {
				for {
					iotChannel <- <-channel
				}
			}()

		default:
			log.Fatalf("Unkown interface requested! %v", pl)
		}
	}

	for {
		select {
		case message := <-iotChannel:
			//Find destination node
			dstnode, err := dbconnection.GetDestination(&message)
			if err != nil {
				log.Fatalf("fogcore: Error in searching for destination node, message: %v", message)
				break
			}
			platform, err := dbconnection.GetPlatform(dstnode.PlatformID)
			if err != nil {
				log.Fatalf("fogcore: Error in searching for platform of destination node, dstnode: %v", dstnode)
				break

			}

			//Send to destination node
			switch dstnode.InfType {
			case iotInterface.Lorawan:
				var opt []grpc.DialOption
				//Get interface options for downlink
				err := cilorawan.ConvertArgsToDownlinkOption(platform.CIArgs["downlink"], &opt)
				if err != nil {
					log.Fatalf("focore: cilorawan: downlink conversion failed: options: %v/n", platform.CIArgs["downlink"])
					break
				}

				//Create client, to send the message
				client, err := cilorawan.NewNetworkClient(context.Background(), platform.CIArgs["downlinkAddress"].(string), opt...)
				if err != nil {
					log.Fatalf("fogcore: cilorawan: creation of newnetworkclient failed! address: %v options: %v\n", platform.CIArgs["downlinkAddress"].(string), opt)
					break
				}
				defer client.Close()
				//Send data with created client
				err = client.SendData(message)
				if err != nil {
					log.Fatalf("fogcore: cilorawan: unable to send message: %v", message)
					break
				}

			case iotInterface.Sixlowpan:
				client, err := cisixlowpan.NewClient("udp6", dstnode.DevID)
				if err != nil {
					log.Fatalf("fogcore: cisixlowpan: unable to create client, destination: %v\n", dstnode.DevID)
					break
				}
				defer client.Close()

				err = client.SendData(message)
				if err != nil {
					log.Fatalf("fogcore: cisixlowpan: unable to send message: %v, error: %v\n", message, err)
					break
				}

			default:
				log.Fatalln("fogcore: Unkown interface of destination node")
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
