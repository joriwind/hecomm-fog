package cilorawan

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"google.golang.org/grpc"

	as "github.com/joriwind/hecomm-fog/api/as"
	"github.com/joriwind/hecomm-fog/iotInterface"
)

func TestStartServer(t *testing.T) {
	type args struct {
		message iotInterface.ComLinkMessage
	}
	comLink := make(chan iotInterface.ComLinkMessage, 5)
	ctx := context.Background()

	asAPI := NewApplicationServerAPI(ctx, comLink)
	go asAPI.StartServer()

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			args: args{
				iotInterface.ComLinkMessage{
					Origin: []byte{0, 0, 0, 0, 0, 0, 0, 0},
					Data:   []byte{1, 2, 3, 4, 5, 6},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Log(tt)
		/*if err := StartServer(tt.args.ctx, tt.args.comLink); (err != nil) != tt.wantErr {
			t.Errorf("%q. StartServer() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}*/
		if err := sendToAsServer(tt.args.message, nil); (err != nil) != tt.wantErr {
			t.Errorf("%q. StartServer() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
		select {
		case m := <-comLink:
			if !testEq(m.Origin, tt.args.message.Origin) && !testEq(m.Data, tt.args.message.Data) {
				t.Errorf("%q. TestStartServer() error: Dit not receive same data as send!, wantErr %v", tt.name, tt.wantErr)
			}
		case <-time.After(time.Second * 1):
			t.Errorf("%q. TestStartServer() error: Timeout, wantErr %v", tt.name, tt.wantErr)
		}
	}
}

func testEq(a, b []byte) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func sendToAsServer(message iotInterface.ComLinkMessage, asDialOptions []grpc.DialOption) error {
	//Create connection to server:
	asDialOptions = append(asDialOptions, grpc.WithInsecure())
	//}
	//host := "192.168.1.1:8000"
	asConn, err := grpc.Dial("localhost:8000", asDialOptions...) //TODO: when close connection?
	defer asConn.Close()
	if err != nil {
		log.Fatalf("application-server (FOG) dial error: %s", err)
		return err
	}
	//defer asConn.Close() //TODO: Do not forget to close connection!
	asClient := as.NewApplicationServerClient(asConn)

	//Send packet!
	publishDataUpReq := as.HandleDataUpRequest{
		AppEUI: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		DevEUI: message.Origin,
		FCnt:   2,
		FPort:  255,
		Data:   message.Data,
		TxInfo: &as.TXInfo{},
	}
	if _, err := asClient.HandleDataUp(context.Background(), &publishDataUpReq); err != nil {
		return fmt.Errorf("publish data up to application-server error: %s", err)
	}
	return nil
}
