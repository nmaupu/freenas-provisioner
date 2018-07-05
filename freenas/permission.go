package freenas

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
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
	resp, err := server.getSlingConnection().Put(endpoint).BodyJSON(p).Receive(nil, nil)
	if err != nil {
		glog.Warningln(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf("Error updating permission - message: %v, status: %d", body, resp.StatusCode))
	}

	return nil
}
