package freenas

import (
	"crypto/tls"
	"fmt"
	"github.com/dghubble/sling"
	"net/http"
)

type FreenasResource interface {
	Delete(server *FreenasServer) error
	CopyFrom(source FreenasResource) error
	Get(server *FreenasServer) error
	Create(server *FreenasServer) error
}

type FreenasServer struct {
	Protocol                 string
	Host, Username, Password string
	Port                     int
	InsecureSkipVerify       bool
	url                      string
}

func NewFreenasServer(protocol string, host string, port int, username, password string, insecure bool) *FreenasServer {
	u := fmt.Sprintf("%s://%s:%d", protocol, host, port)
	return &FreenasServer{
		Protocol:           protocol,
		Host:               host,
		Port:               port,
		Username:           username,
		Password:           password,
		InsecureSkipVerify: insecure,
		url:                u,
	}
}

func (s *FreenasServer) getSlingConnection() *sling.Sling {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: s.InsecureSkipVerify},
	}

	if s.Protocol == "http" {
		tr.TLSClientConfig = nil
	}

	httpClient := &http.Client{Transport: tr}
	return sling.New().Client(httpClient).Base(s.url).SetBasicAuth(s.Username, s.Password)
}
