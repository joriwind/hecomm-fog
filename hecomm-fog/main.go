package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joriwind/hecomm-fog/interfaces"
	"github.com/joriwind/hecomm-fog/interfaces/cilorawan"
)

func main() {
	fmt.Println("Hello world!")
	comLink := make(chan interfaces.ComLinkMessage, 5)
	go cilorawan.Run(context.Background(), comLink)
	select {
	case c := <-comLink:
		log.Printf("Received message from: %v\n", c)
	}

}
