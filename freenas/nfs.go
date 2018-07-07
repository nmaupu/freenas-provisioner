package freenas

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
)

var (
	_ FreenasResource = &NfsShare{}
)

type NfsShare struct {
	Id           int      `json:"id,omitempty"`
	Alldirs      bool     `json:"nfs_alldirs,omitempty"`
	Comment      string   `json:"nfs_comment,omitempty"`
	Hosts        string   `json:"nfs_hosts,omitempty"`
	MapallUser   string   `json:"nfs_mapall_user,omitempty"`
	MapallGroup  string   `json:"nfs_mapall_group,omitempty"`
	MaprootUser  string   `json:"nfs_maproot_user,omitempty"`
	MaprootGroup string   `json:"nfs_maproot_group,omitempty"`
	Network      string   `json:"nfs_network,omitempty"`
	Paths        []string `json:"nfs_paths"`
	Security     []string `json:"nfs_security"`
	Quiet        bool     `json:"nfs_quiet,omitempty"`
	ReadOnly     bool     `json:"nfs_ro,omitempty"`
}

func (n *NfsShare) CopyFrom(source FreenasResource) error {
	src, ok := source.(*NfsShare)
	if ok {
		n.Id = src.Id
		n.Alldirs = src.Alldirs
		n.Comment = src.Comment
		n.Hosts = src.Hosts
		n.MapallUser = src.MapallUser
		n.MapallGroup = src.MapallGroup
		n.MaprootUser = src.MaprootUser
		n.MaprootGroup = src.MaprootGroup
		n.Network = src.Network
		n.Paths = src.Paths
		n.Security = src.Security
		n.Quiet = src.Quiet
		n.ReadOnly = src.ReadOnly
	}

	return errors.New("Cannot copy, src is not a NfsShare")
}

func (n *NfsShare) Get(server *FreenasServer) error {
	if n.Id > 0 {
		endpoint := fmt.Sprintf("/api/v1.0/sharing/nfs/%d/", n.Id)
		var nfs NfsShare
		resp, err := server.getSlingConnection().Get(endpoint).ReceiveSuccess(&nfs)
		if err != nil {
			glog.Warningln(err)
			return err
		}
		defer resp.Body.Close()

		n.CopyFrom(&nfs)
		return nil
	}

	endpoint := "/api/v1.0/sharing/nfs/?limit=1000"
	var shares []NfsShare
	resp, err := server.getSlingConnection().Get(endpoint).ReceiveSuccess(&shares)
	if err != nil {
		glog.Warningln(err)
		return nil
	}
	defer resp.Body.Close()

	for _, share := range shares {
		if share.contains(n.Paths[0]) {
			n.CopyFrom(&share)
			return nil
		}
	}

	// Nothing found
	return errors.New("No NfsShare has been found")
}

func (s *NfsShare) contains(path string) bool {
	for _, p := range s.Paths {
		if p == path {
			return true
		}
	}

	return false
}

func (n *NfsShare) Create(server *FreenasServer) error {
	endpoint := "/api/v1.0/sharing/nfs/"
	var nfs NfsShare
	resp, err := server.getSlingConnection().Post(endpoint).BodyJSON(n).Receive(&nfs, nil)
	if err != nil {
		glog.Warningln(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf("Error creating NFS share for %+v - %v", *n, body))
	}

	n.CopyFrom(&nfs)

	return nil
}

func (n *NfsShare) Delete(server *FreenasServer) error {
	endpoint := fmt.Sprintf("/api/v1.0/sharing/nfs/%d/", n.Id)
	_, err := server.getSlingConnection().Delete(endpoint).Receive(nil, nil)
	if err != nil {
		glog.Warningln(err)
	}
	return err
}
