// wait-for-ironic waits for the Ironic services to be up and running. It is used during the BMO boostrap to
// ensure the availability of all its required dependencies.
package main

import (
	"os"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/baremetal/noauth"
	"github.com/gophercloud/gophercloud/openstack/baremetal/v1/drivers"
	noauthintrospection "github.com/gophercloud/gophercloud/openstack/baremetalintrospection/noauth"
	"github.com/gophercloud/gophercloud/pagination"
)

func main() {
	ironicURL := os.Getenv("IRONIC_ENDPOINT")
	ironicMicroversion := os.Getenv("IRONIC_MICROVERSION")
	inspectorURL := os.Getenv("IRONIC_INSPECTOR_ENDPOINT")

	if ironicURL == "" || inspectorURL == "" {
		fmt.Println("Missing IRONIC_ENDPOINT or IRONIC_INSPECTOR_ENDPOINT env vars")
		os.Exit(1)
	}

	if ironicMicroversion == "" {
		ironicMicroversion = "1.52"
	}

	waitForIronicServices(ironicURL, inspectorURL, ironicMicroversion, 60 * 10)

	fmt.Println("Done!")
}

func waitForIronicServices(ironicURL string, inspectorURL string, ironicMicroversion string, timeout int) {

	waitForIronic(ironicURL, ironicMicroversion, timeout)
	waitForInspector(inspectorURL, timeout)
}

func waitForIronic(ironicURL string, ironicMicroversion string, timeout int) {
	//Get client
	ironic, err := noauth.NewBareMetalNoAuth(noauth.EndpointOpts{
		IronicEndpoint: ironicURL,
	})
	if err != nil {
		fmt.Printf("Unable to configure Ironic endpoint: %s", err.Error())
		os.Exit(1)
	}
	ironic.Microversion = ironicMicroversion

	// Let's poll the API until it's up, or times out.
	duration := time.Duration(timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	done := make(chan struct{})
	go func() {
		log.Printf("[INFO] Waiting for Ironic API...")
		waitForAPI(ctx, ironic)
		log.Printf("[INFO] API successfully connected, waiting for conductor...")
		waitForConductor(ctx, ironic)
		close(done)
	}()

	// Wait for done or time out
	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			fmt.Printf("Unable to contact API: %w", err)
			os.Exit(1)
		}
	case <-done:
	}	
}

func waitForInspector(inspectorURL string, timeout int) {
	//Get client
	inspector, err := noauthintrospection.NewBareMetalIntrospectionNoAuth(noauthintrospection.EndpointOpts{
		IronicInspectorEndpoint: inspectorURL,
	})
	if err != nil {
		fmt.Printf("Unable to configure Inspector endpoint: %s", err.Error())
		os.Exit(1)
	}

	// Let's poll the API until it's up, or times out.
	duration := time.Duration(timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	done := make(chan struct{})
	go func() {
		log.Printf("[INFO] Waiting for Inspector API...")
		waitForAPI(ctx, inspector)
		close(done)
	}()

	// Wait for done or time out
	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			fmt.Printf("Unable to contact Inspector: %w", err)
			os.Exit(1)
		}
	case <-done:
	}
	

}

////////////////////////////////////////////////////////////////////////////

// // Clients stores the client connection information for Ironic and Inspector
// type Clients struct {
// 	ironic    *gophercloud.ServiceClient
// 	inspector *gophercloud.ServiceClient

// 	// Boolean that determines if Ironic API was previously determined to be available, we don't need to try every time.
// 	ironicUp bool

// 	// Boolean that determines we've already waited and the API never came up, we don't need to wait again.
// 	ironicFailed bool

// 	// Mutex so that only one resource being created by terraform checks at a time. There's no reason to have multiple
// 	// resources calling out to the API.
// 	ironicMux sync.Mutex

// 	// Boolean that determines if Inspector API was previously determined to be available, we don't need to try every time.
// 	inspectorUp bool

// 	// Boolean that determines that we've already waited, and inspector API did not come up.
// 	inspectorFailed bool

// 	// Mutex so that only one resource being created by terraform checks at a time. There's no reason to have multiple
// 	// resources calling out to the API.
// 	inspectorMux sync.Mutex

// 	timeout int
// }

// // GetIronicClient returns the API client for Ironic, optionally retrying to reach the API if timeout is set.
// func (c *Clients) GetIronicClient() (*gophercloud.ServiceClient, error) {
// 	// Terraform concurrently creates some resources which means multiple callers can request an Ironic client. We
// 	// only need to check if the API is available once, so we use a mux to restrict one caller to polling the API.
// 	// When the mux is released, the other callers will fall through to the check for ironicUp.
// 	c.ironicMux.Lock()
// 	defer c.ironicMux.Unlock()

// 	// Ironic is UP, or user didn't ask us to check
// 	if c.ironicUp || c.timeout == 0 {
// 		return c.ironic, nil
// 	}

// 	// We previously tried and it failed.
// 	if c.ironicFailed {
// 		return nil, fmt.Errorf("could not contact API: timeout reached")
// 	}

// 	// Let's poll the API until it's up, or times out.
// 	duration := time.Duration(c.timeout) * time.Second
// 	ctx, cancel := context.WithTimeout(context.Background(), duration)
// 	defer cancel()

// 	done := make(chan struct{})
// 	go func() {
// 		log.Printf("[INFO] Waiting for Ironic API...")
// 		waitForAPI(ctx, c.ironic)
// 		log.Printf("[INFO] API successfully connected, waiting for conductor...")
// 		waitForConductor(ctx, c.ironic)
// 		close(done)
// 	}()

// 	// Wait for done or time out
// 	select {
// 	case <-ctx.Done():
// 		if err := ctx.Err(); err != nil {
// 			c.ironicFailed = true
// 			return nil, fmt.Errorf("could not contact API: %w", err)
// 		}
// 	case <-done:
// 	}

// 	c.ironicUp = true
// 	return c.ironic, ctx.Err()
// }

// // GetInspectorClient returns the API client for Ironic, optionally retrying to reach the API if timeout is set.
// func (c *Clients) GetInspectorClient() (*gophercloud.ServiceClient, error) {
// 	// Terraform concurrently creates some resources which means multiple callers can request an Inspector client. We
// 	// only need to check if the API is available once, so we use a mux to restrict one caller to polling the API.
// 	// When the mux is released, the other callers will fall through to the check for inspectorUp.
// 	c.inspectorMux.Lock()
// 	defer c.inspectorMux.Unlock()

// 	if c.inspector == nil {
// 		return nil, fmt.Errorf("no inspector endpoint was specified")
// 	} else if c.inspectorUp || c.timeout == 0 {
// 		return c.inspector, nil
// 	} else if c.inspectorFailed {
// 		return nil, fmt.Errorf("could not contact API: timeout reached")
// 	}

// 	// Let's poll the API until it's up, or times out.
// 	duration := time.Duration(c.timeout) * time.Second
// 	ctx, cancel := context.WithTimeout(context.Background(), duration)
// 	defer cancel()

// 	done := make(chan struct{})
// 	go func() {
// 		log.Printf("[INFO] Waiting for Inspector API...")
// 		waitForAPI(ctx, c.inspector)
// 		close(done)
// 	}()

// 	// Wait for done or time out
// 	select {
// 	case <-ctx.Done():
// 		if err := ctx.Err(); err != nil {
// 			c.ironicFailed = true
// 			return nil, err
// 		}
// 	case <-done:
// 	}

// 	if err := ctx.Err(); err != nil {
// 		c.inspectorFailed = true
// 		return nil, err
// 	}

// 	c.inspectorUp = true
// 	return c.inspector, ctx.Err()
// }

// //NewClients Creates a noauth Ironic client
// func NewClients(ironicURL string, ironicMicroversion string, inspectorURL string, timeout int) (*Clients, error) {
// 	var clients Clients

// 	if ironicURL == "" {
// 		return nil, fmt.Errorf("url is required for ironic provider")
// 	}
// 	log.Printf("[DEBUG] Ironic endpoint is %s", ironicURL)

// 	ironic, err := noauth.NewBareMetalNoAuth(noauth.EndpointOpts{
// 		IronicEndpoint: ironicURL,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	ironic.Microversion = ironicMicroversion
// 	clients.ironic = ironic

// 	if inspectorURL != "" {
// 		log.Printf("[DEBUG] Inspector endpoint is %s", inspectorURL)
// 		inspector, err := noauthintrospection.NewBareMetalIntrospectionNoAuth(noauthintrospection.EndpointOpts{
// 			IronicInspectorEndpoint: inspectorURL,
// 		})
// 		if err != nil {
// 			return nil, fmt.Errorf("could not configure inspector endpoint: %s", err.Error())
// 		}
// 		clients.inspector = inspector
// 	}

// 	clients.timeout = timeout

// 	return &clients, err
// }

// Retries an API forever until it responds.
func waitForAPI(ctx context.Context, client *gophercloud.ServiceClient) {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	// NOTE: Some versions of Ironic inspector returns 404 for /v1/ but 200 for /v1,
	// which seems to be the default behavior for Flask. Remove the trailing slash
	// from the client endpoint.
	endpoint := strings.TrimSuffix(client.Endpoint, "/")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Printf("[DEBUG] Waiting for API to become available...")

			r, err := httpClient.Get(endpoint)
			if err == nil {
				statusCode := r.StatusCode
				r.Body.Close()
				if statusCode == http.StatusOK {
					return
				}
			}

			time.Sleep(5 * time.Second)
		}
	}
}

// Ironic conductor can be considered up when the driver count returns non-zero.
func waitForConductor(ctx context.Context, client *gophercloud.ServiceClient) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Printf("[DEBUG] Waiting for conductor API to become available...")
			driverCount := 0

			drivers.ListDrivers(client, drivers.ListDriversOpts{
				Detail: false,
			}).EachPage(func(page pagination.Page) (bool, error) {
				actual, err := drivers.ExtractDrivers(page)
				if err != nil {
					return false, err
				}
				driverCount += len(actual)
				return true, nil
			})
			// If we have any drivers, conductor is up.
			if driverCount > 0 {
				return
			}

			time.Sleep(5 * time.Second)
		}
	}
}
