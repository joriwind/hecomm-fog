package fogcore

import (
	"github.com/joriwind/hecomm-interface-6lowpan"
)

const (
	confFogcoreAddress string = "192.168.2.123:2000"
	confFogcoreCert    string = "certs/fogcore.pem"
	confFogcoreKey     string = "certs/fogcore-key.key"
)

//SixlowpanPort Used serial connection to communicate with 6LoWPAN
var SixlowpanPort = SixlowpanPortConst

//SixlowpanDebugLevel Debug level
var SixlowpanDebugLevel = SixlowpanDebugLevelConst

const (
	//SixlowpanPortConst  configuration
	SixlowpanPortConst string = "/dev/ttyUSB0"
	//SixlowpanDebugLevelConst configuration
	SixlowpanDebugLevelConst uint8 = sixlowpan.DebugAll
)
