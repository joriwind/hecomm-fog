package main

import (
	"context"
	"strconv"

	"fmt"

	"flag"

	"os"

	"bufio"

	"strings"

	"encoding/json"

	"log"

	"github.com/joriwind/hecomm-fog/dbconnection"
	"github.com/joriwind/hecomm-fog/fogcore"
)

func main() {
	//Flag init
	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Printf("-cert, -key, -address")
	}

	//Checking for flags
	cert := flag.String("cert", "./certs/cert.pem", "The certificate used by TLS listener")
	key := flag.String("key", "./certs/key.unencrypted.pem", "The *unencrypted* key used by TLS listener")
	address := flag.String("address", "localhost:4567", "Server address of TLS listener")
	flag.Parse()

	//Initialise fogcore
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opt := fogcore.Options{
		Hostname:   *address,
		CertServer: *cert,
		KeyServer:  *key,
	}
	fogcore := fogcore.NewFogcore(ctx, opt)
	go func() {
		err := fogcore.Start()
		if err != nil {
			fmt.Printf("Exited with error: %v\n", err)
		} else {
			fmt.Printf("Exited\n")
		}
	}()

	//command line interface of hecomm-fog
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if scanner.Scan() {
			line := scanner.Text()
			//Split line into 2 parts, the command and OPTIONALY data
			command := strings.SplitN(line, " ", 2)
			switch command[0] {
			case "exit":
				cancel()
				return
			case "insert":
				subcommand := strings.SplitN(command[1], " ", 2)
				switch subcommand[0] {
				case "node":
					var node dbconnection.Node
					err := json.Unmarshal(([]byte(subcommand[1])), &node)
					if err != nil {
						fmt.Printf("Not valid Node data: value: %v, error: %v\n", node, err)
						break
					}
					err = dbconnection.InsertNode(&node)
					if err != nil {
						fmt.Printf("Error in inserting node in db: node: %v, error: %v\n", node, err)
						break
					}
					log.Printf("Inserted node in db: value: %v\n", node)

				case "platform":
					var platform dbconnection.Platform
					err := json.Unmarshal(([]byte(subcommand[1])), &platform)
					if err != nil {
						fmt.Printf("Not valid Platform data: value: %v, error: %v\n", platform, err)
						break
					}
					err = dbconnection.InsertPlatform(&platform)
					if err != nil {
						fmt.Printf("Error in inserting platform in db: node: %v, error: %v\n", platform, err)
						break
					}
					log.Printf("Inserted platform in db: value: %v\n", platform)

				case "link":
					var link dbconnection.Link
					err := json.Unmarshal(([]byte(subcommand[1])), &link)
					if err != nil {
						fmt.Printf("Not valid Link data: value: %v, error: %v\n", link, err)
						break
					}
					err = dbconnection.InsertLink(&link)
					if err != nil {
						fmt.Printf("Error in inserting link in db: node: %v, error: %v\n", link, err)
						break
					}
					log.Printf("Inserted link in db: value: %v\n", link)

				default:
					fmt.Printf("Not a valid element: %v\n", subcommand[0])
				}

			case "delete":
				subcommand := strings.SplitN(command[1], " ", 2)
				switch subcommand[0] {
				case "node":
					id, err := strconv.Atoi(subcommand[1])
					if err != nil {
						fmt.Printf("Error in conversion to integer ID: value: %v\n", subcommand[1])
						break
					}
					err = dbconnection.DeleteNode(id)
					if err != nil {
						fmt.Printf("Error in deleting node in db: id: %v, error: %v\n", id, err)
						break
					}
					log.Printf("Deleted node in db: id: %v\n", id)

				case "platform":
					id, err := strconv.Atoi(subcommand[1])
					if err != nil {
						fmt.Printf("Error in conversion to integer ID: value: %v\n", subcommand[1])
						break
					}
					err = dbconnection.DeletePlatform(id)
					if err != nil {
						fmt.Printf("Error in deleting platform in db: id: %v, error: %v\n", id, err)
						break
					}
					log.Printf("Deleted platform in db: id: %v\n", id)

				case "link":
					id, err := strconv.Atoi(subcommand[1])
					if err != nil {
						fmt.Printf("Error in conversion to integer ID: value: %v\n", subcommand[1])
						break
					}
					err = dbconnection.DeleteLink(id)
					if err != nil {
						fmt.Printf("Error in deleting Link in db: id: %v, error: %v\n", id, err)
						break
					}
					log.Printf("Deleted link in db: id: %v\n", id)

				default:
					fmt.Printf("Not a valid element: %v\n", subcommand[0])
				}
			case "get":
				subcommand := strings.SplitN(command[1], " ", 2)
				switch subcommand[0] {
				case "nodes":
					nodes, err := dbconnection.GetNodes()
					if err != nil {
						fmt.Printf("Something went wrong: %v\n", err)
					}
					fmt.Printf("Nodes: %v\n", nodes)

				case "platforms":
					platforms, err := dbconnection.GetPlatforms()
					if err != nil {
						fmt.Printf("Something went wrong: %v\n", err)
					}
					fmt.Printf("Platforms: %v\n", platforms)

				case "links":
					links, err := dbconnection.GetLinks()
					if err != nil {
						fmt.Printf("Something went wrong: %v\n", err)
					}
					fmt.Printf("Links: %v\n", links)

				default:
					fmt.Printf("Not a valid element: %v\n", subcommand[0])
				}

			case "help":
				commands := []string{"insert", "delete", "get"}
				elements := [][]string{{"node", "{\"id\":X,\"devid\":\"XXXX\",\"platformid\":X,\"isprovider\":bool,\"inftype\":X}"},
					{"platform", "{\"id\":X,\"address\":\"XXXX\",\"tlscert\":\"XXX\",\"tlskey\":\"XXXX\",\"citype\":X,\"ciargs\":{}}"},
					{"link", "{\"id\":X,\"provnode\":X,\"reqnode\":X}"}}
				fmt.Println("Available commands:")
				for _, command := range commands {
					switch command {
					case "get":
						fmt.Printf("	%v $ELEMENT(S)\n", command)
					case "delete":
						fmt.Printf("	%v $ELEMENT $ID\n", command)

					default:
						fmt.Printf("	%v $ELEMENT $(OPT)DATA\n", command)
					}
				}
				fmt.Println("Available elements:")
				for _, element := range elements[:] {
					fmt.Printf("	%v	//example: %v\n", element[0], element[1])
				}

			case "":
			default:
				fmt.Printf("Did not understand command: %v\n", command[0])
			}
		}
	}
}
