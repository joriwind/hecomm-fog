package mapping

import (
	"github.com/joriwind/hecomm-api/hecomm"
	"github.com/joriwind/hecomm-fog/dbconnection"
)

//ConvertToLink Search for nodes in the database and create Link element
func ConvertToLink(lc hecomm.LinkContract) (*dbconnection.Link, error) {
	var link dbconnection.Link
	prov, err := dbconnection.FindNode(lc.ProvDevEUI)
	if err != nil {
		return &link, err
	}
	req, err := dbconnection.FindNode(lc.ReqDevEUI)
	if err != nil {
		return &link, err
	}

	link = dbconnection.Link{
		ID:       0, //Undefined link
		ProvNode: prov.ID,
		ReqNode:  req.ID,
	}

	return &link, err
}
