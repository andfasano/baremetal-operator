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

func TestProvisionerIsReady(t *testing.T) {

	driverResponse := `
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

	cases := []struct {
		name      string
		ironic    *testserver.MockServer
		inspector *testserver.MockServer

		expectedIronicCalls    string
		expectedInspectorCalls string
		expectedIsReady        bool
		expectedError          string
	}{
		{
			name:                   "IsReady",
			ironic:                 buildServer(t, "ironic", "{}", driverResponse),
			inspector:              buildServer(t, "ironic-inspector", "{}", driverResponse),
			expectedIronicCalls:    "/v1;/v1/drivers;",
			expectedInspectorCalls: "/v1;",
			expectedIsReady:        true,
		},
		{
			name:                "NoDriversLoaded",
			ironic:              buildServer(t, "ironic", "{}", ""),
			inspector:           testserver.New(t, "inspector"),
			expectedIronicCalls: "/v1;/v1/drivers;",
		},
		{
			name:            "IronicDown",
			inspector:       testserver.New(t, "inspector"),
			expectedIsReady: false,
		},
		{
			name:                "InspectorDown",
			ironic:              buildServer(t, "ironic", "{}", driverResponse),
			expectedIronicCalls: "/v1;/v1/drivers;",
			expectedIsReady:     false,
		},
		{
			name: "IronicNotOk",
			ironic: testserver.New(t, "ironic").
				ErrorResponse("/v1", http.StatusInternalServerError),
			inspector:           testserver.New(t, "inspector"),
			expectedIsReady:     false,
			expectedIronicCalls: "/v1;",
		},
		{
			name: "IronicNotOkAndNotExpected",
			ironic: testserver.New(t, "ironic").
				ErrorResponse("/v1", http.StatusBadGateway),
			inspector:           testserver.New(t, "inspector"),
			expectedIsReady:     false,
			expectedIronicCalls: "/v1;",
		},
		{
			name:   "InspectorNotOk",
			ironic: buildServer(t, "ironic", "{}", driverResponse),
			inspector: testserver.New(t, "inspector").
				ErrorResponse("/v1", http.StatusInternalServerError),
			expectedIsReady:        false,
			expectedIronicCalls:    "/v1;/v1/drivers;",
			expectedInspectorCalls: "/v1;",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.ironic != nil {
				tc.ironic.Start()
				defer tc.ironic.Stop()
			}

			if tc.inspector != nil {
				tc.inspector.Start()
				defer tc.inspector.Stop()
			}

			auth := clients.AuthConfig{Type: clients.NoAuth}

			prov, err := newProvisionerWithSettings(makeHost(), bmc.Credentials{}, nil,
				tc.ironic.Endpoint(), auth, tc.inspector.Endpoint(), auth,
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
