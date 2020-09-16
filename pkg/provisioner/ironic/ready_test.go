package ironic

import (
	"net/http"
	"testing"

	"github.com/metal3-io/baremetal-operator/pkg/bmc"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner/ironic/clients"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner/ironic/testserver"
	"github.com/stretchr/testify/assert"
)

func buildServer(t *testing.T, name string, v1 string, drivers string) *testserver.MockServer {
	return testserver.New(t, name).Response("/v1", v1).
		Response("/v1/drivers", drivers)
}

type ironicMock struct {
	*testserver.MockServer
}

func newIronicMock(t *testing.T) *ironicMock {
	return &ironicMock{
		testserver.New(t, "ironic"),
	}
}

func (i *ironicMock) ready() *ironicMock {
	i.Response("/v1", "{}")
	return i
}

func (i *ironicMock) notReady(errorCode int) *ironicMock {
	i.ErrorResponse("/v1", errorCode)
	return i
}

func (i *ironicMock) addDrivers() *ironicMock {
	i.Response("/v1/drivers", `
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
	`)
	return i
}

type inspectorMock struct {
	*testserver.MockServer
}

func newInspectorMock(t *testing.T) *inspectorMock {
	return &inspectorMock{
		testserver.New(t, "inspector"),
	}
}

func (i *inspectorMock) ready() *inspectorMock {
	i.Response("/v1", "{}")
	return i
}

func (i *inspectorMock) notReady(errorCode int) *inspectorMock {
	i.ErrorResponse("/v1", errorCode)
	return i
}

func TestProvisionerIsReady(t *testing.T) {

	cases := []struct {
		name      string
		ironic    *ironicMock
		inspector *inspectorMock

		expectedIronicCalls    string
		expectedInspectorCalls string
		expectedIsReady        bool
		expectedError          string
	}{
		{
			name:                   "IsReady",
			ironic:                 newIronicMock(t).ready().addDrivers(),
			inspector:              newInspectorMock(t).ready(),
			expectedIronicCalls:    "/v1;/v1/drivers;",
			expectedInspectorCalls: "/v1;",
			expectedIsReady:        true,
		},
		{
			name:                "NoDriversLoaded",
			ironic:              newIronicMock(t).ready(),
			inspector:           newInspectorMock(t).ready(),
			expectedIronicCalls: "/v1;/v1/drivers;",
		},
		{
			name:            "IronicDown",
			inspector:       newInspectorMock(t).ready(),
			expectedIsReady: false,
		},
		{
			name:                "InspectorDown",
			ironic:              newIronicMock(t).ready().addDrivers(),
			expectedIronicCalls: "/v1;/v1/drivers;",
			expectedIsReady:     false,
		},
		{
			name:                "IronicNotOk",
			ironic:              newIronicMock(t).notReady(http.StatusInternalServerError),
			inspector:           newInspectorMock(t).ready(),
			expectedIsReady:     false,
			expectedIronicCalls: "/v1;",
		},
		{
			name:                "IronicNotOkAndNotExpected",
			ironic:              newIronicMock(t).notReady(http.StatusBadGateway),
			inspector:           newInspectorMock(t).ready(),
			expectedIsReady:     false,
			expectedIronicCalls: "/v1;",
		},
		{
			name:                   "InspectorNotOk",
			ironic:                 newIronicMock(t).ready().addDrivers(),
			inspector:              newInspectorMock(t).notReady(http.StatusInternalServerError),
			expectedIsReady:        false,
			expectedIronicCalls:    "/v1;/v1/drivers;",
			expectedInspectorCalls: "/v1;",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ironicEndpoint := "localhost"
			if tc.ironic != nil {
				tc.ironic.Start()
				ironicEndpoint = tc.ironic.Endpoint()
				defer tc.ironic.Stop()
			}

			inspectorEndpoint := "localhost"
			if tc.inspector != nil {
				tc.inspector.Start()
				inspectorEndpoint = tc.inspector.Endpoint()
				defer tc.inspector.Stop()
			}

			auth := clients.AuthConfig{Type: clients.NoAuth}

			prov, err := newProvisionerWithSettings(makeHost(), bmc.Credentials{}, nil,
				ironicEndpoint, auth, inspectorEndpoint, auth,
			)
			if err != nil {
				t.Fatalf("could not create provisioner: %s", err)
			}

			ready, err := prov.IsReady()
			if err != nil {
				t.Fatalf("could not determine ready state: %s", err)
			}

			if tc.ironic != nil {
				assert.Equal(t, tc.expectedIronicCalls, tc.ironic.Requests, "ironic calls")
			}
			if tc.inspector != nil {
				assert.Equal(t, tc.expectedInspectorCalls, tc.inspector.Requests, "inspector calls")
			}

			if tc.expectedError != "" {
				assert.Regexp(t, tc.expectedError, err, "error message")
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.expectedIsReady, ready, "ready flag")
			}
		})
	}
}
