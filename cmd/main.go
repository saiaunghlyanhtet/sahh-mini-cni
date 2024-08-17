package main

import (
	"encoding/json"
	"fmt"

	"sahh-mini-cni/pkg/ipam"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
)

// CNIConfig
type CNIConfig struct {
	types.NetConf
	Subnet string `json:"subnet"`
}

func cmdAdd(args *skel.CmdArgs) error {
	conf := CNIConfig{}
	if err := json.Unmarshal(args.StdinData, &conf); err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	// Allocate an IP address for the container
	result, err := ipam.ExecAdd(conf.IPAM.Type, args.StdinData)
	if err != nil {
		return fmt.Errorf("failed to allocate IP address: %v", err)
	}
	defer func() {
		if clearnupErr := ipam.ExecDel(conf.IPAM.Type, args.StdinData); clearnupErr != nil {
			fmt.Printf("failed to cleanup IP address: %v", clearnupErr)
		}
	}()

	ipconfig := result.(*current.Result)

	return nil
}
