package fogcore

import (
	"github.com/joriwind/hecomm-interface-6lowpan"
)

//ConfFogcoreAddress ...
var ConfFogcoreAddress = "192.168.2.123:2000"

//ConfFogcoreCert ...
var ConfFogcoreCert = "certs/fogcore.cert.pem"

//ConfFogcoreCaCert ...
var ConfFogcoreCaCert = "certs/ca-chain.cert.pem"

//ConfFogcoreKey ...
var ConfFogcoreKey = "private/fogcore.key.pem"

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
