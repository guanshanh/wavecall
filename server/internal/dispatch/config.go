package dispatch

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// Node is one SFU entry in the shared cluster table.
// Dispatch uses Signaling; the SFU process uses PublicIP / UDPPort for ICE.
type Node struct {
	Signaling string `toml:"signaling"` // host:port for WebSocket (returned to clients)
	PublicIP  string `toml:"public_ip"` // ICE host candidate rewrite; empty = no NAT1To1
	UDPPort   int    `toml:"udp_port"`  // media UDPMux port; 0 = leave SFU default / ephemeral
}

type fileConfig struct {
	Dispatch struct {
		Bind string `toml:"bind"`
	} `toml:"dispatch"`
	Nodes map[string]Node `toml:"nodes"`
}

// Config is the shared cluster file: dispatch bind + static node table.
type Config struct {
	Bind  string
	Nodes map[string]Node
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fc fileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		return nil, err
	}
	bind := fc.Dispatch.Bind
	if bind == "" {
		bind = ":18090"
	}
	if len(fc.Nodes) == 0 {
		return nil, fmt.Errorf("dispatch: no nodes in %s", path)
	}
	for k, n := range fc.Nodes {
		if strings.TrimSpace(n.Signaling) == "" {
			return nil, fmt.Errorf("dispatch: empty signaling for node %q in %s", k, path)
		}
	}
	return &Config{Bind: bind, Nodes: fc.Nodes}, nil
}

// LookupNode returns the node with the given config key (e.g. "n1").
func LookupNode(nodes map[string]Node, id string) (Node, error) {
	if id == "" {
		return Node{}, fmt.Errorf("dispatch: empty node id")
	}
	n, ok := nodes[id]
	if !ok {
		return Node{}, fmt.Errorf("dispatch: unknown node %q", id)
	}
	if strings.TrimSpace(n.Signaling) == "" {
		return Node{}, fmt.Errorf("dispatch: empty signaling for node %q", id)
	}
	return n, nil
}
