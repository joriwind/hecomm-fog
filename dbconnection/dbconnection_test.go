package dbconnection

import (
	"testing"

	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func TestPlatform(t *testing.T) {
	type args struct {
		pl *Platform
	}
	f := map[string]interface{}{
		"Name": "Wednesday",
		"Age":  6,
		"Parents": []interface{}{
			"Gomez",
			"Morticia",
		},
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "test1", args: args{&Platform{Address: "localhost/test", CIArgs: f, CIType: 1, TLSCert: "/certs/test.cert", TLSKey: "/certs/key.pem"}}, wantErr: false},
	}
	for _, tt := range tests {
		if err := InsertPlatform(tt.args.pl); (err != nil) != tt.wantErr {
			t.Errorf("%q. InsertPlatform() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			return
		}
		defer func() { //Cleanup function!
			if r := recover(); r != nil {
				fmt.Println("Recovered from error in GetPlatform --> deleting platform")
			}
			if err := DeletePlatform(tt.args.pl.ID); (err != nil) != tt.wantErr {
				t.Errorf("%q. DeletePlatform() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		}()
		if got := GetPlatform(tt.args.pl.ID); !isPlatformEqual(got, tt.args.pl) {
			t.Errorf("%q. GetPlatform() = %v, want %v", tt.name, got, tt.args.pl)
		}

	}
}

//isPlatformEqual compare two Platform structs
func isPlatformEqual(pl *Platform, ref *Platform) bool {
	if pl.Address != ref.Address {
		return false
	}
	if pl.CIType != ref.CIType {
		return false
	}
	if pl.ID != pl.ID {
		return false
	}
	if pl.TLSCert != pl.TLSCert {
		return false
	}
	if pl.TLSKey != pl.TLSKey {
		return false
	}
	if pl.CIArgs["Name"] != ref.CIArgs["Name"] {
		return false
	}

	return true
}
