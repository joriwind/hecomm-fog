package cilorawan

import (
	"context"
	"fmt"
	"log"
	"testing"

	"google.golang.org/grpc"

	as "github.com/joriwind/hecomm-fog/api/as"
	"github.com/joriwind/hecomm-fog/interfaces"
)

func TestStartServer(t *testing.T) {
	type args struct {
		message interfaces.ComLinkMessage
	}
	comLink := make(chan interfaces.ComLinkMessage, 5)
	ctx := context.Background()

	asAPI := NewApplicationServerAPI(ctx, comLink)
	go asAPI.StartServer()

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Log(tt)
		/*if err := StartServer(tt.args.ctx, tt.args.comLink); (err != nil) != tt.wantErr {
			t.Errorf("%q. StartServer() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}*/
	}
}

func sendToAsServer(asDialOptions []grpc.DialOption) error {
	//Create connection to server:
	asDialOptions = append(asDialOptions, grpc.WithInsecure())
	//}
	//host := "192.168.1.1:8000"
	asConn, err := grpc.Dial("localhost:8000", asDialOptions...) //TODO: when close connection?
	if err != nil {
		log.Fatalf("application-server (FOG) dial error: %s", err)
		return err
	}
	//defer asConn.Close() //TODO: Do not forget to close connection!
	asClient := as.NewApplicationServerClient(asConn)

	//Send packet!
	publishDataUpReq := as.HandleDataUpRequest{
		AppEUI: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		DevEUI: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		FCnt:   2,
		FPort:  255,
		Data:   []byte{1, 2, 3, 4, 5},
		TxInfo: &as.TXInfo{},
	}
	if _, err := asClient.HandleDataUp(context.Background(), &publishDataUpReq); err != nil {
		return fmt.Errorf("publish data up to application-server error: %s", err)
	}
	return nil
}
