package dbconnection

import "database/sql"
import _ "github.com/go-sql-driver/mysql" //Mysql database
import "errors"
import "encoding/json"

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
	dbsource string = "hecomm:password@tcp(localhost:5555)/dbname?charset=utf8"
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
	stmt, err := db.Prepare("INSERT platform SET address=?, citype=?, ciargs=?")
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
	res, err := stmt.Exec(pl.Address, pl.CIType, string(jsonBytes))
	if err != nil {
		return err
	}

	//Check response for confirmation insertion
	i, err := res.LastInsertId()
	if err != nil {
		return err
	}
	pl.ID = int(i)
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

	row := stmt.QueryRow(id)
	var citype int
	var address, tlscert, tlskey, ciargs string
	row.Scan(&id, &citype, &address, &tlscert, &tlskey, &ciargs)
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(ciargs), &data); err != nil {
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
