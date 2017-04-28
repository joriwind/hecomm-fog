package fogcore

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"
)

//Fogcore Struct
type Fogcore struct {
	ctx           context.Context
	opt           FogcoreOptions
	ctxInterfaces []context.Context
}

type ci struct {
	ctx     context.Context
	ciType  string
	comLink chan []byte
}

type FogcoreOptions struct {
	Hostname   string
	CertServer string
	KeyServer  string
}

//NewFogcore Create new fogcore module
func NewFogcore(ctx context.Context, opt FogcoreOptions) *Fogcore {
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

	for {
		select {
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
