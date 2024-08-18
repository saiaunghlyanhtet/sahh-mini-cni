package network

import (
	"fmt"
	"os/exec"
)

// SetupInterface sets up the network interface inside the container
func SetupInterface(ifName, ip, subnet string) error {
	// Create a veth pair
	if err := exec.Command("ip", "link", "add", ifName, "type", "veth", "peer", "name", "cni0").Run(); err != nil {
		return fmt.Errorf("failed to create veth pair: %v", err)
	}

	// Assign IP address to the container interface
	if err := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/%s", ip, subnet), "dev", ifName).Run(); err != nil {
		return fmt.Errorf("failed to assign IP address: %v", err)
	}

	// Bring up the container interface
	if err := exec.Command("ip", "link", "set", ifName, "up").Run(); err != nil {
		return fmt.Errorf("failed to bring up interface: %v", err)
	}

	return nil
}

// SetupRouting sets up direct routing for pod-to-pod and pod-to-node communication
func SetupRouting(ip, subnet string) error {
	// Add a route for pod-to-pod communication without NAT
	if err := exec.Command("ip", "route", "add", subnet, "dev", "cni0").Run(); err != nil {
		return fmt.Errorf("failed to add pod-to-pod route: %v", err)
	}

	// Add a route for node-to-pod communication
	if err := exec.Command("ip", "route", "add", ip, "dev", "cni0").Run(); err != nil {
		return fmt.Errorf("failed to add node-to-pod route: %v", err)
	}

	// Ensure no NAT is applied by setting up iptables rules or disabling masquerade
	if err := exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-s", subnet, "!", "-o", "cni0", "-j", "MASQUERADE").Run(); err != nil {
		return fmt.Errorf("failed to disable NAT: %v", err)
	}

	return nil
}

// TeardownInterface removes the network interface from the container
func TeardownInterface(ifName string) error {
	// Remove the network interface
	cmd := exec.Command("ip", "link", "delete", ifName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete interface: %v", err)
	}

	return nil
}
