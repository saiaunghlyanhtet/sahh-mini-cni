package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/saiaunghlyanhtet/sahh-mini-cni/pkg/ipam"
	"github.com/saiaunghlyanhtet/sahh-mini-cni/pkg/network"
)

// CNIConfig represents the CNI configuration structure
type CNIConfig struct {
	types.NetConf
	Subnet string `json:"subnet"`
}

// IPConfig represents the IP configuration
type IPConfig struct {
	Version string `json:"version"`
	Address string `json:"address"`
}

// Result represents the result of a CNI configuration
type Result struct {
	IPs []*IPConfig `json:"ips"`
}

// adding a container to the network
func cmdAdd(stdinData []byte) (*Result, error) {
	var conf CNIConfig
	if err := json.Unmarshal(stdinData, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %v", err)
	}

	confFile := string(stdinData)
	// Allocate an IP address for the container
	result, err := ipam.ExecAdd(confFile)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP address: %v", err)
	}
	defer func() {
		if cleanupErr := ipam.ExecDel(confFile, result.IP); cleanupErr != nil {
			log.Printf("failed to cleanup IP address: %v", cleanupErr)
		}
	}()

	ip := result.IP

	// Set up the network interface inside the container with the allocated IP
	if err := network.SetupInterface(conf.Name, ip, conf.Subnet); err != nil {
		return nil, fmt.Errorf("failed to set up network interface: %v", err)
	}

	// Set up direct routing for pod-to-pod and pod-to-node communication
	if err := network.SetupRouting(ip, conf.Subnet); err != nil {
		return nil, fmt.Errorf("failed to set up routing: %v", err)
	}

	ipNet := &net.IPNet{
		IP:   net.ParseIP(ip),
		Mask: net.CIDRMask(24, 32),
	}

	_, maskSizeInBits := ipNet.Mask.Size()

	ipConfig := &IPConfig{
		Version: "4",
		Address: fmt.Sprintf("%s/%d", ipNet.IP.String(), maskSizeInBits),
	}

	return &Result{
		IPs: []*IPConfig{ipConfig},
	}, nil
}

// removing a container from the network
func cmdDel(stdinData []byte) error {
	var conf CNIConfig
	if err := json.Unmarshal(stdinData, &conf); err != nil {
		return fmt.Errorf("failed to parse configuration: %v", err)
	}

	confFile := string(stdinData)
	// Release the IP allocation
	if err := ipam.ExecDel(confFile, conf.Name); err != nil {
		return fmt.Errorf("failed to deallocate IP: %v", err)
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s [add|del]", os.Args[0])
	}

	action := os.Args[1]
	stdinData, err := os.ReadFile("/dev/stdin")
	if err != nil {
		log.Fatalf("Failed to read stdin: %v", err)
	}

	switch action {
	case "add":
		result, err := cmdAdd(stdinData)
		if err != nil {
			log.Fatalf("cmdAdd failed: %v", err)
		}

		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Fatalf("Failed to marshal result: %v", err)
		}
		fmt.Println(string(resultJSON))

	case "del":
		if err := cmdDel(stdinData); err != nil {
			log.Fatalf("cmdDel failed: %v", err)
		}

	default:
		log.Fatalf("Unknown action %s", action)
	}
}
