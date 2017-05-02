package dbconnection

import "database/sql"
import _ "github.com/go-sql-driver/mysql" //Mysql database
import "errors"
import "encoding/json"
import "fmt"
import "log"

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
func GetPlatform(id int) *Platform {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	stmt, err := db.Prepare("SELECT * FROM platform WHERE id=?")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(id)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {

		var citype int
		var address, tlscert, tlskey, ciargs string
		if err := rows.Scan(&id, &address, &tlscert, &tlskey, &citype, &ciargs); err != nil {
			panic(err)
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(ciargs), &data); err != nil {
			fmt.Printf("Platform from query: %v, %v, %v, %v, %v, %v\n", id, citype, address, tlscert, tlskey, ciargs)
			panic(err)
		}
		return &Platform{
			ID:      id,
			Address: address,
			TLSCert: tlscert,
			TLSKey:  tlskey,
			CIType:  citype,
			CIArgs:  data,
		}
	}
	return &Platform{}
}

//GetPlatforms Retrieve all platforms
func GetPlatforms() []*Platform {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	stmt, err := db.Prepare("SELECT * FROM platform")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var platforms []*Platform

	for rows.Next() {

		var id, citype int
		var address, tlscert, tlskey, ciargs string
		if err := rows.Scan(&id, &address, &tlscert, &tlskey, &citype, &ciargs); err != nil {
			panic(err)
		}
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(ciargs), &data); err != nil {
			fmt.Printf("Platform from query: %v, %v, %v, %v, %v, %v\n", id, citype, address, tlscert, tlskey, ciargs)
			panic(err)
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
	return platforms
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

//GetNode Retrieve node via device identifier
func GetNode(devID []byte) *Node {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	stmt, err := db.Prepare("SELECT * FROM node WHERE devid=?")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	row := stmt.QueryRow(devID)
	var id, platformid, inftype int
	var devid []byte
	var isprovider bool
	row.Scan(&id, &devid, &platformid, &isprovider, &inftype)
	return &Node{
		ID:         id,
		DevID:      devid,
		PlatformID: platformid,
		IsProvider: isprovider,
		InfType:    inftype,
	}
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
func GetLink(nodeID int) *Link {
	db, err := sql.Open(dbDriver, dbsource)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	stmt, err := db.Prepare("SELECT id, provnode, reqnode FROM link WHERE provnode=? OR reqnode=?")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	row := stmt.QueryRow(nodeID, nodeID)
	var id, provnode, reqnode int
	row.Scan(&id, &provnode, &reqnode)
	return &Link{
		ID:       id,
		ProvNode: provnode,
		ReqNode:  reqnode,
	}
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
