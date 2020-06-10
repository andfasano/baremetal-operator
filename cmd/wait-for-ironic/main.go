// wait-for-ironic waits for the Ironic services to be up and running. It is used during the BMO boostrap to
// ensure the availability of all its required dependencies.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: wait-for-ironic <ironic URI> <inspector URI> <node UUID>")
		return
	}

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

	clients, err := NewClients(ironicURL, ironicMicroversion, inspectorURL, 0)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Waiting for Ironic...")
	_, err = clients.GetIronicClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Waiting for Ironic Inspector...")
	_, err = clients.GetInspectorClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}
