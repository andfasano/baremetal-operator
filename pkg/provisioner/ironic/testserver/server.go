package testserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func New(t *testing.T, name string) *MockServer {
	return &MockServer{
		t:    t,
		name: name,
	}
}

type MockServer struct {
	t         *testing.T
	name      string
	Requests  string
	server    *httptest.Server
	drivers   string
	errorCode int
}

func (m *MockServer) SetErrorCode(code int) *MockServer {
	m.errorCode = code

	return m
}

func (m *MockServer) AddDrivers() *MockServer {
	m.drivers = `
	{
		"drivers": [{
			"hosts": [
			  "master-2.ostest.test.metalkube.org"
			],
			"links": [
			  {
				"href": "http://[fd00:1101::3]:6385/v1/drivers/fake-hardware",
				"rel": "self"
			  },
			  {
				"href": "http://[fd00:1101::3]:6385/drivers/fake-hardware",
				"rel": "bookmark"
			  }
			],
			"name": "fake-hardware"
		}]
	}
	`
	return m
}

func (m *MockServer) Endpoint() string {
	if m == nil || m.server == nil {
		// The consumer of this method expects something valid, but
		// won't use it if m is nil.
		return "https://ironic.test/v1/"
	}
	response := m.server.URL + "/v1/"
	m.t.Logf("%s: endpoint: %s/", m.name, response)
	return response
}

func (m *MockServer) logRequest(r *http.Request, msg string) {
	m.t.Logf("%s: %s %s", m.name, msg, r.URL)
	m.Requests += r.RequestURI + ";"
}

func (m *MockServer) handleNoResponse(w http.ResponseWriter, r *http.Request) {
	m.logRequest(r, "no response")
	if m.errorCode != 0 {
		http.Error(w, "An error", m.errorCode)
		return
	}
}

func (m *MockServer) handleDrivers(w http.ResponseWriter, r *http.Request) {
	m.logRequest(r, "drivers")
	if m.errorCode != 0 {
		http.Error(w, "An error", m.errorCode)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, m.drivers)
}

func (m *MockServer) Start() *MockServer {

	mux := http.NewServeMux()
	mux.HandleFunc("/", m.handleNoResponse)
	mux.HandleFunc("/v1", m.handleNoResponse)
	mux.HandleFunc("/v1/drivers", m.handleDrivers)

	m.server = httptest.NewServer(mux)

	return m
}

func (m *MockServer) Stop() {
	m.server.Close()
}
