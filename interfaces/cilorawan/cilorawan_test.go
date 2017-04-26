package cilorawan

import (
	"context"
	"testing"

	"github.com/joriwind/hecomm-fog/interfaces"
)

func TestStartServer(t *testing.T) {
	type args struct {
		message interfaces.ComLinkMessage
	}
	comLink := make(chan interfaces.ComLinkMessage, 5)
	ctx := context.Background()

	if err := StartServer(ctx, comLink); err != nil {
		t.Errorf("StartServer() error = %v", err)
	}

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
