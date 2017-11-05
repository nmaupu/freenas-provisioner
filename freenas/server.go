package freenas

import (
	"crypto/tls"
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
	Host, Username, Password string
	Port                     int
	InsecureSkipVerify       bool
	url                      string
}

func NewFreenasServer(host, port, username, password string, insecure bool) *FreenasServer {
	u := fmt.Sprintf("https://%s:%d", host, port)
	return &FreenasServer{
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
	httpClient := &http.Client{Transport: tr}
	return sling.New().Client(httpClient).Base(s.url).SetBasicAuth(s.Username, s.Password)
}
