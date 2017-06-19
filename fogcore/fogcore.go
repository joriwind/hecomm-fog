package fogcore

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"log"
	"net"

	"time"

	"encoding/json"

	"io"

	"fmt"

	"github.com/joriwind/hecomm-api/hecomm"
	"github.com/joriwind/hecomm-fog/dbconnection"
	"github.com/joriwind/hecomm-fog/iotInterface"
	"github.com/joriwind/hecomm-fog/iotInterface/cilorawan"
	"github.com/joriwind/hecomm-fog/iotInterface/cisixlowpan"
	"github.com/joriwind/hecomm-fog/mapping"
)

//Fogcore Struct
type Fogcore struct {
	ctx          context.Context
	opt          Options
	controlCH    chan controlCHMessage
	ciCommonCH   chan iotInterface.ComLinkMessage
	ciCollection []ci
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

/*
 *	State of hecomm protocol
 * Linked: state of hecomm protocol: 0 if not found partner, 1 if found partner
 */
type linkState struct {
	LC       hecomm.LinkContract
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
		fogcore.opt.Hostname = confFogcoreAddress
	case opt.KeyServer == "":
		fogcore.opt.KeyServer = confFogcoreKey
	case opt.CertServer == "":
		fogcore.opt.CertServer = confFogcoreCert
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
	//f.ciCollection = make([]ci, len(platforms))
	f.ciCommonCH = make(chan iotInterface.ComLinkMessage, 20)
	//Create the communication to the iot interface thread

	//Startup already known interfaces
	for _, pl := range platforms {
		channel := make(chan iotInterface.ComLinkMessage, 5)
		ctx, cancel := context.WithCancel(f.ctx)
		platform := pl //Map variable, else last value used!
		face := ci{Platform: &platform, Channel: channel, Ctx: ctx, Cancel: cancel}
		f.ciCollection = append(f.ciCollection, face)
		f.startInterface(&f.ciCollection[len(f.ciCollection)-1])
	}

	for {
		select {
		case cm := <-f.controlCH:
			if err := f.executeCommand(&cm.Message); err != nil {
				log.Printf("Error in executeCommand! controlMessage: %v\n", err)
				cm.ResponseCH <- false
				//return nil
			}
			cm.ResponseCH <- true
		case clm := <-f.ciCommonCH:
			if err := f.handleCIMessage(&clm); err != nil {
				log.Printf("Error in handleCIMessage! message: %v\n", clm)
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

	caCert, err := ioutil.ReadFile(f.opt.CertServer)
	if err != nil {
		log.Fatalf("cacert error: %v\n", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
	}
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

			log.Printf("fogcore: accepted TLS connection from %s", conn.RemoteAddr())
			tlscon, ok := conn.(*tls.Conn)
			if ok {
				state := tlscon.ConnectionState()
				for _, v := range state.PeerCertificates {
					log.Print(x509.MarshalPKIXPublicKey(v.PublicKey))
				}
				log.Printf("New connection from: %v\n", conn.RemoteAddr())
				go f.handleTLSConn(conn)
			}
		case <-f.ctx.Done():
			return nil
		}
	}

}

func (f *Fogcore) handleTLSConn(conn net.Conn) {
	buf := make([]byte, 2048)
	defer conn.Close()
	for {
		//Read
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF { //Check if connection was closed by remote
				log.Printf("Connection closed by remote: %v\n", conn.RemoteAddr())
				return
			}
			log.Printf("fogcore: handleTLSConn: error: %v\n", err)
			return
		}
		//var m tlsMessage
		//err = json.Unmarshal(buf[0:n], m)
		m, err := hecomm.GetMessage(buf[:n])
		if err != nil {
			log.Printf("fogcore: handleTLSConn: NewMessage: error: %v\n", err)
			return
		}
		log.Printf("Hecomm message received: FPort: %v, remote: %v\n", m.FPort, conn.RemoteAddr())

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
				log.Printf("fogcore: handleTLSConn: GetCommand error: %v\n", err)
				return
			}
			resp := make(chan bool, 1)
			cchm := controlCHMessage{
				Message:    *cm,
				ResponseCH: resp,
			}
			//Sending command to main routine, waiting for answer, also getting ready to close connection
			f.controlCH <- cchm
			response := <-resp
			log.Printf("DBcommand resulted in: %v\n", response)
			rsp, err := hecomm.NewResponse(response)
			if err != nil {
				log.Printf("fogcore: handleTLSConn: getbytes of response, error: %v\n", err)
				return
			}
			//Writing answer to client
			conn.Write(rsp)
			//Stop connection
			break
		default:
			log.Printf("Unexpected FPort: %v\n", m.FPort)
		}

	}
}

func (ls *linkState) handleLinkProtocol(sP *hecomm.Message) {
	//Buffers
	var message *hecomm.Message
	var err error
	var rcv []byte
	rcvOrigFromReq := true

	//Init
	message = sP
	chReq := make(chan []byte, 10)
	chProv := make(chan []byte, 10)
	chError := make(chan error)

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

	//Keep running while protocol is active
	for {

		//Do action depending on type of message
		switch message.FPort {

		case hecomm.FPortLinkReq:
			//TODO: check requesting node and platform, in db?
			lc, err := message.GetLinkContract()
			if err != nil {
				log.Printf("Unable to formulate LinkContract from message: %v, error: %v\n", string(message.Data), err)
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
					log.Fatalf("Failed to formulate %v response, error: %v\n", false, err)
				}
				ls.ReqConn.Write(bytes)
				return
			}

			//Ready to find partner
			ls.LC = *lc

			//Locating a possible provider node
			tmpProvnode, err := dbconnection.FindAvailableProviderNode(lc.InfType)
			if err != nil {
				log.Fatalf("Error in finding provider node in DB: InfType: %v, error: %v\n", lc.InfType, err)
			}
			if tmpProvnode.ID == 0 {
				log.Printf("fogcore: handleLinkProtocol: Dit not find suitable provider node! link request: %v\n", string(sP.Data))
				//Sending failed response
				bytes, err := hecomm.NewResponse(false)
				if err != nil {
					log.Fatalf("Failed to formulate false response, error: %v\n", err)
				}
				ls.ReqConn.Write(bytes)
				return
			}

			platform, err := dbconnection.GetPlatform(tmpProvnode.PlatformID)
			if err != nil {
				log.Fatalf("Failed to retrieve platform from DB, platform ID: %v, error: %v\n", tmpProvnode.ID, err)
			}

			//Setup tls connection to provider platform
			conf := &tls.Config{
				InsecureSkipVerify: true,
			}
			ls.ProvConn, err = tls.Dial("tcp", platform.Address, conf)
			if err != nil {
				log.Printf("Could not reach provider platform: %v\n", err)
				//TODO: connection not available
				bytes, err := hecomm.NewResponse(false)
				if err != nil {
					log.Fatalf("Failed to formuate false response, error: %v\n", err)
				}
				ls.ReqConn.Write(bytes)
				return
			}
			defer ls.ProvConn.Close()

			//Send contract for node to provider platform
			ls.LC.ProvDevEUI = []byte(tmpProvnode.DevID)
			bytes, err := ls.LC.GetBytes()
			if err != nil {
				log.Fatalf("Failed to compile linkcontract into bytes, linkcontract: %v, error: %v\n", ls.LC, err)
				return
			}
			ls.ProvConn.Write(bytes)

			//Tunnel data from provider to channel provider
			go func(ch chan []byte, chError chan error, buf []byte) {
				for {
					n, err := ls.ProvConn.Read(buf)
					if err != nil {
						chError <- err
						return
					}
					ch <- buf[:n]
				}
			}(chProv, chError, ls.BufProv)

		case hecomm.FPortLinkState:
			//Depending on origin of data send to the other
			if rcvOrigFromReq {
				ls.ProvConn.Write(message.Data)
			} else {
				ls.ReqConn.Write(message.Data)
			}

		case hecomm.FPortLinkSet:
			//TODO:Check if memorised LC is similar to received Linkcontract
			lc, err := message.GetLinkContract()
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: unvalid data set packet: %v, error: %v\n", string(message.Data), err)
				return
			}
			//If status linked and received from requester --> send contract to provider
			if lc.Linked == true && rcvOrigFromReq {
				//Send contract to provider
				ls.LC.Linked = true
				bytes, err := ls.LC.GetBytes()
				if err != nil {
					log.Fatalf("fogcore: handleLinkProtocol: linkcontract to bytes, error: %v\n", err)
					return

				}
				ls.ProvConn.Write(bytes)
			}

		case hecomm.FPortResponse:
			rsp, err := message.GetResponse()
			if err != nil {
				log.Fatalf("fogcore: handleLinkProtocol: invalid response message: %v, error %v\n", message, err)
			}
			switch ls.LC.Linked {
			case false:
				//Found valid partner or not!
				if rsp.OK && !rcvOrigFromReq { //Response to request for prov node

					//Sending linkcontract to requester
					bytes, err := ls.LC.GetBytes()
					if err != nil {
						log.Fatalf("fogcore: handleLinkProtocol: ok tslResponse, error: %v\n", err)
						return
					}
					ls.ReqConn.Write(bytes)

				} else {
					//TODO: in case of not valid response, search for other provider!!
					log.Fatalf("fogcore: handleLinkProtocol: NOT OK response, what to do? State: %v\n", ls)
				}
			case true:
				//Connection is set and key was generated!
				if rsp.OK && !rcvOrigFromReq {

					//Sending OK response to requester
					bytes, err := hecomm.NewResponse(true)
					if err != nil {
						log.Fatalf("fogcore: handleLinkProtocol: ok tslResponse, error: %v\n", err)
						return
					}
					ls.ReqConn.Write(bytes)
					link, err := mapping.ConvertToLink(ls.LC)
					if err != nil {
						log.Fatalf("fogcore: handleLinkProtocol: could not convert contract to link: contract: %v, error: %v\n", ls.LC, err)
					}
					err = dbconnection.InsertLink(link)
					if err != nil {
						log.Fatalf("fogcore: handleLinkProtocol: could not insert link: contract: %v, error: %v\n", link, err)
					}
					//Link is set!
					return
				}
				//TODO: in case of not valid response, search for other provider
				log.Fatalf("fogcore: handleLinkProtocol: NOT OK response, what to do? State: %v\n", ls)

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

		//Translate packet
		message, err = hecomm.GetMessage(rcv)
		if err != nil {
			log.Fatalf("fogcore: handleLinkProtocol: unable to unmarshal linkmessage: %v\n", err)
		}
	}
}

//executeCommand Handle the control messages
func (f *Fogcore) executeCommand(command *hecomm.DBCommand) error {
	log.Printf("Executing DBCommand: EType: %v, Insert: %v\n", command.EType, command.Insert)
	switch command.EType {
	case hecomm.ETypePlatform: //Start new platform
		//Unravel data from command packet into platform element
		var element hecomm.DBCPlatform
		err := json.Unmarshal(command.Data, &element)
		if err != nil {
			return err
		}

		platform := dbconnection.Platform{
			Address: element.Address,
			CIType:  int(element.CI),
		}

		//Depending on insert bool, insert or delete
		switch command.Insert {
		case true:
			//Check if interface already exists?
			for index, ci := range f.ciCollection {
				//TODO update if address isn't correct: ci.Platform.Address == platform.Address &&
				if ci.Platform.CIType == platform.CIType {
					log.Printf("Execute command: Communication interface already present and active")
					//If address do not match, update
					if ci.Platform.Address != platform.Address {
						log.Printf("Updating platform with new information: %v\n", platform)
						var newPlatform dbconnection.Platform
						newPlatform = *ci.Platform
						newPlatform.Address = platform.Address
						err := dbconnection.UpdatePlatform(&newPlatform)
						if err != nil {
							return err
						}
						//Stop old platform interface
						ci.Cancel()

						//Start new
						ctx, cancel := context.WithCancel(f.ctx)
						ci.Cancel = cancel
						ci.Ctx = ctx
						ci.Platform = &newPlatform
						//Channel is reused

						f.startInterface(&f.ciCollection[index])
					}
					return nil
				}

			}

			channel := make(chan iotInterface.ComLinkMessage, 5)
			ctx, cancel := context.WithCancel(f.ctx)
			//Add new interface to end of collection
			f.ciCollection = append(f.ciCollection, ci{Platform: &platform, Channel: channel, Ctx: ctx, Cancel: cancel})
			//Startup last added interface
			f.startInterface(&f.ciCollection[len(f.ciCollection)-1])

			//Add to db
			err := dbconnection.InsertPlatform(&platform)
			if err != nil {
				return err
			}
			log.Printf("New platform inserted: %v\n", platform)

		case false: //Stop a platform
			for index, intface := range f.ciCollection {
				if intface.Platform.Address == platform.Address && intface.Platform.CIType == platform.CIType {
					//Remove from db
					err = dbconnection.DeletePlatform(intface.Platform.ID)
					if err != nil {
						return err
					}

					//Delete while preserving order
					//append the slice part before the element with all the elements after the specific element
					f.ciCollection = append(f.ciCollection[:index], f.ciCollection[index+1:]...)

				}
			}
			log.Printf("Platform Deleted: %v\n", platform)

		}

	case hecomm.ETypeNode:
		var node dbconnection.Node
		var element hecomm.DBCNode
		var platformID int
		err := json.Unmarshal(command.Data, &element)
		if err != nil {
			return err
		}
		pls, err := dbconnection.GetPlatforms()
		if err != nil {
			return err
		}
		for _, pl := range pls {
			if pl.Address == element.PlAddress {
				if pl.CIType == int(element.PlType) {
					platformID = pl.ID
					break
				}
			}
		}
		if platformID == 0 {
			return fmt.Errorf("Could not find platform for node, origin: %v, type: %v", element.PlAddress, element.PlType)
		}

		node = dbconnection.Node{
			DevID:      string(element.DevEUI),
			InfType:    element.InfType,
			IsProvider: element.IsProvider,
			PlatformID: platformID,
		}

		//Depending on insert bool, insert or delete
		switch command.Insert {
		case true:
			err := dbconnection.InsertNode(&node)
			if err != nil {
				return err
			}

		case false:
			err := dbconnection.DeleteNode(node.ID)
			if err != nil {
				return err
			}
		}
		log.Printf("New node inserted %v\n", node)

	default:
		return fmt.Errorf("fogcore: executeCommand: unexpected EType: %v", command.EType)

	}

	return nil
}

//startInterface Start listening on new interface
func (f *Fogcore) startInterface(iot *ci) error {
	//Starup the listening routines for all platforms in the database

	//Depending on type, create iot interface routine
	switch iot.Platform.CIType {
	case int(hecomm.CILorawan):

		//args := (grpc.ServerOption)pl.CIArgs
		lorawanapi := cilorawan.NewApplicationServerAPI(iot.Ctx, iot.Channel)
		log.Println("Starting LoRaWAN interface!")
		//Start the cilorawan
		go func() {
			if err := lorawanapi.StartServer(); err != nil {
				log.Printf("Something went wrong in interface: %v; error: %v", iot, err)
			}
		}()
		//Tunnel the communication to common channel -- easy access in main loop
		go func() {
			for {
				f.ciCommonCH <- <-iot.Channel
			}
		}()

	case int(hecomm.CISixlowpan):
		//Create the communication to the iot interface thread

		sixlowpanServer := cisixlowpan.NewServer(iot.Ctx, iot.Channel)
		log.Println("Starting 6LoWPAN interface!")
		//Start the cilorawan
		go func() {
			if err := sixlowpanServer.Start(); err != nil {
				log.Printf("Something went wrong in interface: %v; error: %v", iot, err)
			}
		}()
		//Tunnel the communication to common channel -- easy access in main loop
		go func() {
			for {
				f.ciCommonCH <- <-iot.Channel
			}
		}()

	default:
		return fmt.Errorf("Unkown interface requested! %v", iot)
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
	case int(hecomm.CILorawan):

		//Create client, to send the message
		client, err := cilorawan.NewNetworkClient(context.Background(), cilorawan.ConfNSAddress)
		if err != nil {
			log.Fatalf("fogcore: cilorawan: creation of newnetworkclient failed! address: %v\n", "")
			return err
		}
		defer client.Close()
		//Send data with created client
		err = client.SendData(*clm)
		if err != nil {
			log.Fatalf("fogcore: cilorawan: unable to send message: %v", *clm)
			return err
		}

	case int(hecomm.CISixlowpan):
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
