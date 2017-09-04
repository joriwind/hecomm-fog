package fogcore

import (
	"github.com/joriwind/hecomm-interface-6lowpan"
)

const (
	confFogcoreAddress string = "192.168.2.123:2000"
	confFogcoreCert    string = "certs/fogcore.pem"
	confFogcoreKey     string = "certs/fogcore-key.key"
)

const (
	//SLIPPort  configuration
	SLIPPort string = "/dev/ttyUSB0"
	//SLIPDebug configuration
	SLIPDebug uint8 = sixlowpan.DebugPacket
)
