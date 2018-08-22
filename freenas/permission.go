package freenas

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
)

type Permission struct {
	Path  string `json:"mp_path"`
	Acl   string `json:"mp_acl"`
	Mode  string `json:"mp_mode"`
	User  string `json:"mp_user"`
	Group string `json:"mp_group"`
}

func (p *Permission) Put(server *FreenasServer) error {
	endpoint := "/api/v1.0/storage/permission/"
	var e interface{}
	resp, err := server.getSlingConnection().Put(endpoint).BodyJSON(p).Receive(nil, &e)
	if err != nil {
		glog.Warningln(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := json.Marshal(e)
		return errors.New(fmt.Sprintf("Error updating permission - message: %v, status: %d", string(body), resp.StatusCode))
	}

	return nil
}
