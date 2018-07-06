package freenas

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"path/filepath"
)

var (
	_ FreenasResource = &Dataset{}
)

type Dataset struct {
	Avail          int64  `json:"avail,omitempty"`
	Mountpoint     string `json:"mountpoint,omitempty"`
	Name           string `json:"name"`
	Pool           string `json:"pool"`
	Recordsize     int64  `json:"recordsize,omitempty"`
	Refquota       int64  `json:"refquota,omitempty"`
	Refreservation int64  `json:"refreservation,omitempty"`
	Refer          int64  `json:"refer,omitempty"`
	Used           int64  `json:"used,omitempty"`
	Comments       string `json:"comments,omitempty"`
}

func (d *Dataset) String() string {
	return filepath.Join(d.Pool, d.Name)
}

func (d *Dataset) CopyFrom(source FreenasResource) error {
	src, ok := source.(*Dataset)
	if ok {
		d.Avail = src.Avail
		d.Mountpoint = src.Mountpoint
		d.Name = src.Name
		d.Pool = src.Pool
		d.Recordsize = src.Recordsize
		d.Refquota = src.Refquota
		d.Refreservation = src.Refreservation
		d.Refer = src.Refer
		d.Used = src.Used
		d.Comments = src.Comments
	}

	return errors.New("Cannot copy, src is not a Dataset")
}

func (d *Dataset) Get(server *FreenasServer) error {
	endpoint := fmt.Sprintf("/api/v1.0/storage/dataset/%s/", d.Name)
	var dataset Dataset
	resp, err := server.getSlingConnection().Get(endpoint).ReceiveSuccess(&dataset)
	if err != nil {
		glog.Warningln(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf("Error getting dataset \"%s\" - message: %v, status: %d", d.Name, body, resp.StatusCode))
	}

	d.CopyFrom(&dataset)

	return nil
}

func (d *Dataset) Create(server *FreenasServer) error {
	parent, dsName := filepath.Split(d.Name)
	endpoint := fmt.Sprintf("/api/v1.0/storage/dataset/%s", parent)

	// rewrite Name attribute to support crazy api semantics
	d.Name = dsName

	resp, err := server.getSlingConnection().Post(endpoint).BodyJSON(d).Receive(nil, nil)

	// rewrite Name attribute to support crazy api semantics
	d.Name = filepath.Join(parent, dsName)

	if err != nil {
		glog.Warningln(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf("Error creating dataset \"%s\" - message: %v, status: %d", d.Name, body, resp.StatusCode))
	}

	return nil
}

func (d *Dataset) Delete(server *FreenasServer) error {
	endpoint := fmt.Sprintf("/api/v1.0/storage/dataset/%s/", d.Name)
	resp, err := server.getSlingConnection().Delete(endpoint).Receive(nil, nil)
	if err != nil {
		glog.Warningln(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf("Error deleting dataset \"%s\" - %v", d.Name, body))
	}

	return nil
}
