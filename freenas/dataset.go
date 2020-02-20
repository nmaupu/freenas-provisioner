package freenas

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"path/filepath"
	"strconv"
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
	Quota          int64  `json:"quota,omitempty"`
	Reservation    int64  `json:"reservation,omitempty"`
	Refquota       int64  `json:"refquota,omitempty"`
	Refreservation int64  `json:"refreservation,omitempty"`
	Refer          int64  `json:"refer,omitempty"`
	Used           int64  `json:"used,omitempty"`
	Comments       string `json:"comments,omitempty"`
}

func (d *Dataset) MarshalJSON() ([]byte, error) {
	data := &struct {
		Avail          int64  `json:"avail,omitempty"`
		Mountpoint     string `json:"mountpoint,omitempty"`
		Name           string `json:"name"`
		Pool           string `json:"pool"`
		Recordsize     int64  `json:"recordsize,omitempty"`
		Quota          string `json:"quota,omitempty"`
		Reservation    string `json:"reservation,omitempty"`
		Refquota       string `json:"refquota,omitempty"`
		Refreservation string `json:"refreservation,omitempty"`
		Refer          int64  `json:"refer,omitempty"`
		Used           int64  `json:"used,omitempty"`
		Comments       string `json:"comments,omitempty"`
	}{
		Avail:      d.Avail,
		Mountpoint: d.Mountpoint,
		Name:       d.Name,
		Pool:       d.Pool,
		Recordsize: d.Recordsize,
		Refer:      d.Refer,
		Used:       d.Used,
		Comments:   d.Comments,
	}

	if d.Quota > 0 {
		data.Quota = strconv.FormatInt(d.Quota, 10) + "b"
	}

	if d.Reservation > 0 {
		data.Reservation = strconv.FormatInt(d.Reservation, 10) + "b"
	}

	if d.Refquota > 0 {
		data.Refquota = strconv.FormatInt(d.Refquota, 10) + "b"
	}

	if d.Refreservation > 0 {
		data.Refreservation = strconv.FormatInt(d.Refreservation, 10) + "b"
	}

	return json.Marshal(data)
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
		d.Quota = src.Quota
		d.Reservation = src.Reservation
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
	var e interface{}
	resp, err := server.getSlingConnection().Get(endpoint).Receive(&dataset, &e)
	if err != nil {
		glog.Warningln(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := json.Marshal(e)
		return errors.New(fmt.Sprintf("Error getting dataset \"%s\" - message: %v, status: %d", d.Name, string(body), resp.StatusCode))
	}

	d.CopyFrom(&dataset)

	return nil
}

func (d *Dataset) Create(server *FreenasServer) error {
	parent, dsName := filepath.Split(d.Name)
	endpoint := fmt.Sprintf("/api/v1.0/storage/dataset/%s", parent)
	var dataset Dataset
	var e interface{}

	// rewrite Name attribute to support crazy api semantics
	d.Name = dsName

	resp, err := server.getSlingConnection().Post(endpoint).BodyJSON(d).Receive(&dataset, &e)

	// rewrite Name attribute to support crazy api semantics
	d.Name = filepath.Join(parent, dsName)

	if err != nil {
		glog.Warningln(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := json.Marshal(e)
		return errors.New(fmt.Sprintf("Error creating dataset \"%s\" - message: %v, status: %d", d.Name, string(body), resp.StatusCode))
	}

	d.CopyFrom(&dataset)

	return nil
}

func (d *Dataset) Delete(server *FreenasServer) error {
	endpoint := fmt.Sprintf("/api/v1.0/storage/dataset/%s/", d.Name)
	var e interface{}
	resp, err := server.getSlingConnection().Delete(endpoint).Receive(nil, &e)
	if err != nil {
		glog.Warningln(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		body, _ := json.Marshal(e)
		return errors.New(fmt.Sprintf("Error deleting dataset \"%s\" - %v", d.Name, string(body)))
	}

	return nil
}
