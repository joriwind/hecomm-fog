package fogcore

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"

	"time"

	"encoding/json"

	"github.com/joriwind/hecomm-fog/dbconnection"
	"github.com/joriwind/hecomm-fog/hecomm"
	"github.com/joriwind/hecomm-fog/iotInterface"
	"github.com/joriwind/hecomm-fog/iotInterface/cilorawan"
	"github.com/joriwind/hecomm-fog/iotInterface/cisixlowpan"
	"google.golang.org/grpc"
)

//Fogcore Struct
type Fogcore struct {
	ctx          context.Context
	opt          Options
	controlCH    chan controlCHMessage
	ciCommonCH   chan iotInterface.ComLinkMessage
	ciCollection []*ci
}

//Options Defines possible options to pass along with Fogcore object
type Options struct {
	Hostname   string
	CertServer string
	KeyServer  string
}

type ci struct {
	Platform *dbconnection.Platform
	Channel  chan iotInterface.ComLinkMessage
	Ctx      context.Context
	Cancel   func()
}

type controlCHMessage struct {
	Message    hecomm.DBCommand
	ResponseCH chan bool
}

type linkState struct {
	State    int
	ReqConn  net.Conn
	ProvConn net.Conn
	BufReq   []byte
	BufProv  []byte
	Ctx      context.Context
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
	f.controlCH = make(chan controlCHMessage, 20)
	go f.listenOnTLS()

	//Startup already known platforms
	platforms, err := dbconnection.GetPlatforms()
	if err != nil {
		log.Fatalf("fogcore: something went wrong in retrieving interfaces: %v", err)
	}
	//Create access to the will be routines of iot interfaces
	f.ciCollection = make([]*ci, len(platforms))
	f.ciCommonCH = make(chan iotInterface.ComLinkMessage, 20)
	//Create the communication to the iot interface thread

	//Startup already known interfaces
	for index, pl := range platforms {
		channel := make(chan iotInterface.ComLinkMessage, 5)
		ctx, cancel := context.WithCancel(f.ctx)
		f.ciCollection[index] = &ci{Platform: pl, Channel: channel, Ctx: ctx, Cancel: cancel}
		f.startInterface(f.ciCollection[index])
	}

	for {
		select {
		case cm := <-f.controlCH:
			if err := f.executeCommand(&cm.Message); err != nil {
				log.Fatalf("Error in executeCommand! controlMessage: %v\n", cm)
				cm.ResponseCH <- false
			}
			cm.ResponseCH <- true
		case clm := <-f.ciCommonCH:
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
			go f.handleTLSConn(conn)
		case <-f.ctx.Done():
			return nil
		}
	}

}

func (f *Fogcore) handleTLSConn(conn net.Conn) {
	buf := make([]byte, 2048)
	for {
		//Read
		n, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("fogcore: handleTLSConn: error: %v\n", err)
		}
		//var m tlsMessage
		//err = json.Unmarshal(buf[0:n], m)
		m, err := hecomm.NewMessage(buf[:n])
		if err != nil {
			log.Fatalf("fogcore: handleTLSConn: NewMessage: error: %v\n", err)
		}

		//Detect control message, is boolean 'Link' true or false?
		switch m.FPort {
		case 10:
			bufProv := make([]byte, 2048)
			ctx, cancel := context.WithTimeout(f.ctx, time.Minute*5)
			defer cancel()
			ls := linkState{
				ReqConn: conn,
				BufReq:  buf,
				BufProv: bufProv,
				Ctx:     ctx,
			}
			ls.handleLinkProtocol(m)

		case 0:
			//Unmarshal the data part of hecomm message as command
			cm, err := m.GetCommand()
			if err != nil {
				log.Fatalf("fogcore: handleTLSConn: GetCommand error: %v\n", err)
			}
			resp := make(chan bool, 1)
			cchm := controlCHMessage{
				Message:    *cm,
				ResponseCH: resp,
			}
			//Sending command to main routine, waiting for answer, also getting ready to close connection
			f.controlCH <- cchm
			defer conn.Close()
			response := <-resp
			rsp, err := (&hecomm.Response{OK: response}).GetBytes()
			if err != nil {
				log.Fatalf("fogcore: handleTLSConn: getbytes of response, error: %v\n", err)
				return
			}
			//Writing answer to client
			conn.Write(rsp)
			//Stop connection
			return
		}

	}
}

func (ls *linkState) handleLinkProtocol(sP *hecomm.Message) {
	var message *hecomm.Message
	var err error
	var rcv []byte
	rcvOrigFromReq := true
	rcv = sP.Data
	chReq := make(chan []byte, 10)
	chProv := make(chan []byte, 10)
	chError := make(chan error)

	//Close channel when done
	defer ls.ReqConn.Close()

	//Tunnel data from requester to channel requester
	go func(ch chan []byte, chError chan error, buf []byte) {
		for {
			n, err := ls.ReqConn.Read(buf)
			if err != nil {
				chError <- err
				return
			}
			ch <- buf[:n]
		}
	}(chReq, chError, ls.BufReq)

	for {

		message, err = hecomm.NewMessage(rcv)
		if err != nil {
			log.Fatalf("fogcore: handleLinkProtocol: unable to unmarshal linkmessage: %v\n", err)
		}

		//Do action depending on type of message
		switch message.FPort {

		case hecomm.FPortLinkReq:
			//TODO: check requesting node, in db?
			lc, err := message.GetLinkContract()
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: unvalid link request packet: %v, error: %v\n", string(message.Data), err)
				break
			}

			//Check if requester node is in the db
			reqNode, err := dbconnection.FindNode(lc.ReqDevEUI)
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: error in locating requesting node: %v, error %v\n", lc, err)
			}
			//If not valid id
			if reqNode.ID == 0 {
				log.Printf("fogcore: handleLinkProtocol: dit not find requesting node in db: %v\n", message)
				bytes, err := hecomm.NewResponse(false)
				if err != nil {
					log.Fatalf("fogcore: handleLinkProtocol: failed response, error: %v\n", err)
				}
				ls.ReqConn.Write(bytes)
				return
			}

			//Locating a possible provider node
			node, err := dbconnection.FindAvailableProviderNode(lc.InfType)
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: error in locating provider node: %v, error: %v\n", message, err)
			}
			if node.ID == 0 {
				log.Fatalf("fogcore: handleLinkProtocol: Dit not find suitable provider node! link request: %v\n", string(sP.Data))
				//Sending failed response
				bytes, err := hecomm.NewResponse(false)
				if err != nil {
					log.Fatalf("fogcore: handleLinkProtocol: failed response, error: %v\n", err)
				}
				ls.ReqConn.Write(bytes)
				return
			}
			platform, err := dbconnection.GetPlatform(node.ID)
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: getplatform of available node failed: %v\n", err)
			}

			//Setup tls connection to provider platform
			conf := &tls.Config{
			//InsecureSkipVerify: true,
			}
			ls.ProvConn, err = tls.Dial("tcp", platform.Address, conf)
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: could not reach provider platform: %v\n", err)
				//TODO: connection not available
				return
			}
			defer ls.ProvConn.Close()

			//Send request for node to provider platform
			lc.ProvDevEUI = []byte(node.DevID)
			bytes, err := lc.GetBytes()
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: failed to send node request to provider platform, error: %v\n", err)
				return
			}
			ls.ProvConn.Write(bytes)

		case hecomm.FPortLinkState:
			//Depending on origin of data send to the other
			if rcvOrigFromReq {
				ls.ProvConn.Write(message.Data)
			} else {
				ls.ReqConn.Write(message.Data)
			}

		case hecomm.FPortLinkSet:
			lc, err := message.GetLinkContract()
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: unvalid data set packet: %v, error: %v\n", string(message.Data), err)
				return
			}
			link, err := lc.ConvertToLink()
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: could not convert contract to link: contract: %v, error: %v\n", lc, err)
			}
			err = dbconnection.InsertLink(link)
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: could not insert link: contract: %v, error: %v\n", link, err)
			}

		case hecomm.FPortResponse:
			rsp, err := message.GetResponse()
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: invalid response message: %v, error %v\n", message, err)
			}
			//TODO: response in different cases, depending on state of connection
			if rsp.OK && !rcvOrigFromReq { //Response to request for prov node

				//Sending OK response to requester
				bytes, err := hecomm.NewResponse(true)
				if err != nil {
					log.Fatalf("fogcore: handleLinkProtocol: ok tslResponse, error: %v\n", err)
					return
				}
				ls.ReqConn.Write(bytes)

				//Tunnel data from requester to channel requester
				go func(ch chan []byte, chError chan error, buf []byte) {
					for {
						n, err := ls.ReqConn.Read(buf)
						if err != nil {
							chError <- err
							return
						}
						ch <- buf[:n]
					}
				}(chProv, chError, ls.BufProv)
			} else {
				//TODO: in case of not valid response, search for other provider
				log.Fatalf("fogcore: handleLinkProtocol: NOT OK response, what to do?\n")
			}

		default:
			log.Fatalf("fogcore: handleLinkProtocol: unexpected FPort: %v\n", message.FPort)

		}

		//Wait for either packet from requester or data provider
		select {
		case rcv = <-chReq:
			rcvOrigFromReq = true

		case rcv = <-chProv:
			rcvOrigFromReq = false

		case err := <-chError:
			log.Fatalf("fogcore: handleLinkProtocol: received error from a channel: %v\n", err)
			return

		case <-ls.Ctx.Done():
			log.Fatalf("fogcore: handleLinkProtocol: context ended linkState: %v\n", ls)
			return
		}
	}
}

//executeCommand Handle the control messages
func (f *Fogcore) executeCommand(command *hecomm.DBCommand) error {
	switch command.EType {
	case hecomm.ETypePlatform: //Start new platform
		//Unravel data from command packet into platform element
		var platform dbconnection.Platform
		err := json.Unmarshal(command.Data, platform)
		if err != nil {
			log.Fatalf("fogcore: executeCommand: unable to unmarshal platform from bytes: data: %v, err: %v\n", command.Data, err)
		}
		//Depending on insert bool, insert or delete
		switch command.Insert {
		case true:
			channel := make(chan iotInterface.ComLinkMessage, 5)
			ctx, cancel := context.WithCancel(f.ctx)
			//Add new interface to end of collection
			f.ciCollection = append(f.ciCollection, &ci{Platform: &platform, Channel: channel, Ctx: ctx, Cancel: cancel})
			//Startup last added interface
			f.startInterface(f.ciCollection[len(f.ciCollection)-1])

		case false: //Stop a platform
			for index, intface := range f.ciCollection {
				if intface.Platform.ID == platform.ID {
					//Delete while preserving order
					//append the slice part before the element with all the elements after the specific element
					f.ciCollection = append(f.ciCollection[:index], f.ciCollection[index+1:]...)
				}
			}
			//TODO: delete from db and certs?
		}

	case hecomm.ETypeNode:
		var node dbconnection.Node
		err := json.Unmarshal(command.Data, node)
		if err != nil {
			log.Fatalf("fogcore: executeCommand: unable to unmarshal node from bytes: data: %v, err: %v\n", command.Data, err)
		}
		//Depending on insert bool, insert or delete
		switch command.Insert {
		case true:
			err := dbconnection.InsertNode(&node)
			if err != nil {
				log.Fatalf("fogcore: executeCommand: unable to insert node into db: node: %v, error: %v\n", node, err)
			}

		case false:
			err := dbconnection.DeleteNode(node.ID)
			if err != nil {
				log.Fatalf("fogcore: executeCommand: unable to delete node from db: node: %v, error: %v\n", node, err)
			}
		}

	default:
		log.Fatalf("fogcore: executeCommand: unexpected EType: %v\n", *command)

	}

	return nil
}

//startInterface Start listening on new interface
func (f *Fogcore) startInterface(iot *ci) error {
	//Starup the listening routines for all platforms in the database

	//Depending on type, create iot interface routine
	switch iot.Platform.CIType {
	case iotInterface.Lorawan:

		//Convert general mapping to cilorawan Server Options
		var args []grpc.ServerOption
		cilorawan.ConvertArgsToUplinkOptions(iot.Platform.CIArgs["uplink"], &args)

		//args := (grpc.ServerOption)pl.CIArgs
		lorawanapi := cilorawan.NewApplicationServerAPI(iot.Ctx, iot.Channel, args...)

		//Start the cilorawan
		go func() {
			if err := lorawanapi.StartServer(); err != nil {
				log.Fatalf("Something went wrong in interface: %v; error: %v", iot, err)
			}
		}()
		//Tunnel the communication to common channel -- easy access in main loop
		go func() {
			for {
				f.ciCommonCH <- <-iot.Channel
			}
		}()

	case iotInterface.Sixlowpan:
		//Create the communication to the iot interface thread

		var args cisixlowpan.ServerOptions
		cisixlowpan.ConvertArgsToUplinkOptions(iot.Platform.CIArgs["uplink"], &args)

		sixlowpanServer := cisixlowpan.NewServer(iot.Ctx, iot.Channel, args)

		//Start the cilorawan
		go func() {
			if err := sixlowpanServer.Start(); err != nil {
				log.Fatalf("Something went wrong in interface: %v; error: %v", iot, err)
			}
		}()
		//Tunnel the communication to common channel -- easy access in main loop
		go func() {
			for {
				f.ciCommonCH <- <-iot.Channel
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
