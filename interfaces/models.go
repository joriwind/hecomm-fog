package interfaces

import "time"

//ComLinkMessage message structure to be used to communicate with fogCore
type ComLinkMessage struct {
	InterfaceType int
	Origin        []byte
	Destination   []byte
	TimeReceived  time.Time
	Data          []byte
}

//Defines the InterfaceTypes of InterfaceType integer of ComLinkMessage
const (
	Lorawan int = 1 + iota
	Sixlowpan
)
