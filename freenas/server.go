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
	Url, Username, Password string
	InsecureSkipVerify      bool
}

func NewFreenasServer(url, username, password string, insecure bool) *FreenasServer {
	return &FreenasServer{
		Url:                url,
		Username:           username,
		Password:           password,
		InsecureSkipVerify: insecure,
	}
}

func (s *FreenasServer) getSlingConnection() *sling.Sling {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: s.InsecureSkipVerify},
	}
	httpClient := &http.Client{Transport: tr}
	return sling.New().Client(httpClient).Base(s.Url).SetBasicAuth(s.Username, s.Password)
}
