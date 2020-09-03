package ironic

import (
	"net/http"
	"testing"

	"github.com/metal3-io/baremetal-operator/pkg/bmc"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner/ironic/clients"
	"github.com/metal3-io/baremetal-operator/pkg/provisioner/ironic/testserver"
	"github.com/stretchr/testify/assert"
)

func TestProvisionerIsReady(t *testing.T) {

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
			name:      "IsReady",
			ironic:    testserver.New(t, "ironic").AddDrivers(),
			inspector: testserver.New(t, "inspector"),

			expectedIronicCalls:    "/v1;/v1/drivers;",
			expectedInspectorCalls: "/v1;",
			expectedIsReady:        true,
		},
		{
			name:      "NoDriversLoaded",
			ironic:    testserver.New(t, "ironic"),
			inspector: testserver.New(t, "inspector"),

			expectedIronicCalls: "/v1;/v1/drivers;",
		},
		{
			name:      "IronicDown",
			inspector: testserver.New(t, "inspector"),

			expectedIsReady: false,
		},
		{
			name:   "InspectorDown",
			ironic: testserver.New(t, "ironic").AddDrivers(),

			expectedIronicCalls: "/v1;/v1/drivers;",

			expectedIsReady: false,
		},
		{
			name:      "IronicNotOk",
			ironic:    testserver.New(t, "ironic").SetErrorCode(http.StatusInternalServerError),
			inspector: testserver.New(t, "inspector"),

			expectedIsReady: false,

			expectedIronicCalls: "/v1;",
		},
		{
			name:      "IronicNotOkAndNotExpected",
			ironic:    testserver.New(t, "ironic").SetErrorCode(http.StatusBadGateway),
			inspector: testserver.New(t, "inspector"),

			expectedIsReady: false,

			expectedIronicCalls: "/v1;",
		},
		{
			name:      "InspectorNotOk",
			ironic:    testserver.New(t, "ironic").AddDrivers(),
			inspector: testserver.New(t, "inspector").SetErrorCode(http.StatusInternalServerError),

			expectedIsReady: false,

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
				assert.Equal(t, tc.expectedIronicCalls, tc.ironic.Requests)
			}
			if tc.inspector != nil {
				assert.Equal(t, tc.expectedInspectorCalls, tc.inspector.Requests)
			}

			if tc.expectedError != "" {
				assert.Regexp(t, tc.expectedError, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.expectedIsReady, ready)
			}
		})
	}
}
