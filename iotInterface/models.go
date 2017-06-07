package iotInterface

import (
	"time"

	"github.com/joriwind/hecomm-api/hecomm"
)

//ComLinkMessage message structure to be used to communicate with fogCore
type ComLinkMessage struct {
	InterfaceType hecomm.CIType
	Origin        []byte
	Destination   []byte
	TimeReceived  time.Time
	Data          []byte
}
