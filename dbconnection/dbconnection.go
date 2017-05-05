package dbconnection

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joriwind/hecomm-fog/iotInterface"
)

//Mysql database

//Platform Model of a platform in the mysql database
type Platform struct {
	ID      int
	Address string
	TLSCert string
	TLSKey  string
	CIType  int
	CIArgs  map[string]interface{}
}

//Node Model of a Node in the mysql database
type Node struct {
	ID         int
	DevID      []byte
	PlatformID int
	IsProvider bool
	InfType    int
}

//Link Model of a Link stored in db between two communicating nodes
type Link struct {
	ID       int
	ProvNode int
	ReqNode  int
}

const (
	dbsource string = "hecomm:hecomm@tcp(localhost:3306)/hecomm?charset=utf8"
	dbDriver string = "mysql"
)

//InsertPlatform Insert a new platform in the mysql database
func InsertPlatform(pl *Platform) error {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return err
	}
	defer db.Close()

	//Prepare insert query
	stmt, err := db.Prepare("INSERT platform SET address=?, citype=?, ciargs=?, tlscert=?, tlskey=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	//Create JSON string
	jsonBytes, err := json.Marshal(pl.CIArgs)
	if err != nil {
		return err
	}

	//Execute insert
	res, err := stmt.Exec(pl.Address, pl.CIType, string(jsonBytes), pl.TLSCert, pl.TLSKey)
	if err != nil {
		return err
	}

	//Check response for confirmation insertion
	i, err := res.LastInsertId()
	if err != nil {
		return err
	}
	pl.ID = int(i)
	log.Printf("Inserted platform: Address: %v, ciargs: %v, citype: %v, id: %v, tlscert: %v, tlskey: %v", pl.Address, pl.CIArgs, pl.CIType, pl.ID, pl.TLSCert, pl.TLSKey)
	return nil

}

//UpdatePlatform Update a platform row in the database
func UpdatePlatform(pl *Platform) error {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("UPDATE platform SET address=?, citype=?, ciargs=? WHERE id=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	//Create JSON string
	jsonBytes, err := json.Marshal(pl.CIArgs)
	if err != nil {
		return err
	}

	res, err := stmt.Exec(pl.Address, pl.CIType, string(jsonBytes), pl.ID)
	if err != nil {
		return err
	}
	if i, _ := res.RowsAffected(); i != 1 {
		return errors.New("dbconnection: failed to update platform")
	}
	return nil
}

//GetPlatform Retrieve platform via platform id
func GetPlatform(id int) (*Platform, error) {
	var platform Platform

	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return &platform, err
	}
	defer db.Close()
	stmt, err := db.Prepare("SELECT * FROM platform WHERE id=?")
	if err != nil {
		return &platform, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(id)
	if err != nil {
		return &platform, err
	}
	defer rows.Close()
	for rows.Next() {

		var citype int
		var address, tlscert, tlskey, ciargs string
		if err := rows.Scan(&id, &address, &tlscert, &tlskey, &citype, &ciargs); err != nil {
			return &platform, err
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(ciargs), &data); err != nil {
			fmt.Printf("Platform from query: %v, %v, %v, %v, %v, %v\n", id, citype, address, tlscert, tlskey, ciargs)
			return &platform, err
		}
		platform = Platform{
			ID:      id,
			Address: address,
			TLSCert: tlscert,
			TLSKey:  tlskey,
			CIType:  citype,
			CIArgs:  data,
		}
		return &platform, nil
	}
	return &platform, err
}

//GetPlatforms Retrieve all platforms
func GetPlatforms() ([]*Platform, error) {
	var platforms []*Platform
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return platforms, err
	}
	defer db.Close()
	stmt, err := db.Prepare("SELECT * FROM platform")
	if err != nil {
		return platforms, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return platforms, err
	}
	defer rows.Close()

	for rows.Next() {

		var id, citype int
		var address, tlscert, tlskey, ciargs string
		if err := rows.Scan(&id, &address, &tlscert, &tlskey, &citype, &ciargs); err != nil {
			return platforms, err
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(ciargs), &data); err != nil {
			fmt.Printf("Platform from query: %v, %v, %v, %v, %v, %v\n", id, citype, address, tlscert, tlskey, ciargs)
			return platforms, err
		}
		platforms = append(platforms, &Platform{
			ID:      id,
			Address: address,
			TLSCert: tlscert,
			TLSKey:  tlskey,
			CIType:  citype,
			CIArgs:  data,
		})
	}
	return platforms, nil
}

//DeletePlatform Delete platform via platform id
func DeletePlatform(id int) error {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("DELETE FROM platform WHERE id=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)

	if err != nil {
		return err
	}
	if i, err := res.RowsAffected(); i != 1 {
		if err != nil {
			return err
		}
		return fmt.Errorf("DeletePlatform, Delete result did not equal 1:%v", i)
	}
	return nil
}

//InsertNode Insert a node into the database
func InsertNode(n *Node) error {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("INSERT node SET devid=?, platformid=?, isprovider=?, inftype=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(n.DevID, n.PlatformID, n.IsProvider, n.InfType)
	if err != nil {
		return err
	}
	i, err := res.LastInsertId()
	if err != nil {
		return err
	}
	n.ID = int(i)
	return nil
}

//UpdateNode Update a node from the database
func UpdateNode(n *Node) error {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("UPDATE node SET devid=?, platformid=?, isprovider=?, inftype=? WHERE id=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(n.DevID, n.PlatformID, n.IsProvider, n.InfType, n.ID)
	if err != nil {
		return err
	}
	if i, _ := res.RowsAffected(); i != 1 {
		return errors.New("dbconnection: failed to insert node")
	}
	return nil
}

//DeleteNode Delete node via id
func DeleteNode(id int) error {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("DELETE FROM node WHERE id=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)

	if err != nil {
		return err
	}
	if i, err := res.RowsAffected(); i != 1 {
		if err != nil {
			return err
		}
		return fmt.Errorf("DeleteNode, Delete result did not equal 1:%v", i)
	}
	return nil
}

//FindNode Retrieve node via device identifier
func FindNode(devID []byte) (*Node, error) {
	var node Node
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return &node, err
	}
	defer db.Close()
	stmt, err := db.Prepare("SELECT * FROM node WHERE devid=?")
	if err != nil {
		return &node, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(devID)
	var id, platformid, inftype int
	var devid []byte
	var isprovider bool
	row.Scan(&id, &devid, &platformid, &isprovider, &inftype)
	node = Node{
		ID:         id,
		DevID:      devid,
		PlatformID: platformid,
		IsProvider: isprovider,
		InfType:    inftype,
	}
	return &node, err
}

//FindAvailableProviderNode Locate a node that is still available to transfer the required data
func FindAvailableProviderNode(infType int) (*Node, error) {
	var node Node
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return &node, err
	}
	defer db.Close()
	stmt, err := db.Prepare("SELECT * FROM node LEFT JOIN link ON link.provnode = node.id WHERE node.inftype=? AND link.id is null AND node.isprovider = 1")
	if err != nil {
		return &node, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(infType)
	var id, platformid, inftype int
	var devid []byte
	var isprovider bool
	row.Scan(&id, &devid, &platformid, &isprovider, &inftype)
	node = Node{
		ID:         id,
		DevID:      devid,
		PlatformID: platformid,
		IsProvider: isprovider,
		InfType:    inftype,
	}
	return &node, err
}

//GetNode Retrieve node via device identifier
func GetNode(ID int) (*Node, error) {
	var node Node
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return &node, err
	}
	defer db.Close()
	stmt, err := db.Prepare("SELECT * FROM node WHERE id=?")
	if err != nil {
		return &node, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(ID)
	var id, platformid, inftype int
	var devid []byte
	var isprovider bool
	row.Scan(&id, &devid, &platformid, &isprovider, &inftype)
	node = Node{
		ID:         id,
		DevID:      devid,
		PlatformID: platformid,
		IsProvider: isprovider,
		InfType:    inftype,
	}
	return &node, err
}

//InsertLink Insert a link into the database
func InsertLink(l *Link) error {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("INSERT link SET provnode=?, reqnode=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(l.ProvNode, l.ReqNode)
	if err != nil {
		return err
	}
	i, err := res.LastInsertId()
	if err != nil {
		return err
	}
	l.ID = int(i)
	return nil
}

//UpdateLink Update a link in the database
func UpdateLink(l *Link) error {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("UPDATE link SET provnode=?, reqnode=? WHERE id=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(l.ProvNode, l.ReqNode, l.ID)
	if err != nil {
		return err
	}
	if i, _ := res.RowsAffected(); i != 1 {
		return errors.New("dbconnection: failed to insert node")
	}
	return nil
}

//GetLink Retrieve via one of both's node ID
func GetLink(nodeID int) (*Link, error) {
	var link Link
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return &link, err
	}
	defer db.Close()
	stmt, err := db.Prepare("SELECT id, provnode, reqnode FROM link WHERE provnode=? OR reqnode=?")
	if err != nil {
		return &link, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(nodeID, nodeID)
	var id, provnode, reqnode int
	row.Scan(&id, &provnode, &reqnode)
	link = Link{
		ID:       id,
		ProvNode: provnode,
		ReqNode:  reqnode,
	}
	return &link, err
}

//DeleteLink Delete link via id
func DeleteLink(id int) error {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("DELETE FROM link WHERE id=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(id)

	if err != nil {
		return err
	}
	if i, err := res.RowsAffected(); i != 1 {
		if err != nil {
			return err
		}
		return fmt.Errorf("DeleteLink, Delete result did not equal 1:%v", i)
	}
	return nil
}

//GetDestination fill message with destination and return the destination node
func GetDestination(message *iotInterface.ComLinkMessage) (*Node, error) {

	srcnode, err := FindNode(message.Origin)
	if err != nil {
		return nil, err
	}

	link, err := GetLink(srcnode.ID)
	if err != nil {
		return nil, err
	}
	var dstnode *Node
	switch srcnode.ID {
	case link.ProvNode:
		dstnode, err = GetNode(link.ReqNode)
	case link.ReqNode:
		dstnode, err = GetNode(link.ProvNode)
	}
	if err != nil {
		return nil, err
	}

	message.Destination = dstnode.DevID

	return dstnode, nil
}
