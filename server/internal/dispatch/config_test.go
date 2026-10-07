package dispatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigStructuredNodes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cluster.toml")
	content := `
[dispatch]
bind = ":8091"

[nodes.n1]
signaling = "127.0.0.1:18080"
public_ip = "203.0.113.10"
udp_port = 18081

[nodes.n2]
signaling = "sfu2.example.com:18080"
public_ip = "203.0.113.20"
udp_port = 18081
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Bind != ":8091" {
		t.Fatalf("bind %q", cfg.Bind)
	}
	n1, err := LookupNode(cfg.Nodes, "n1")
	if err != nil {
		t.Fatal(err)
	}
	if n1.Signaling != "127.0.0.1:18080" || n1.PublicIP != "203.0.113.10" || n1.UDPPort != 18081 {
		t.Fatalf("n1 %+v", n1)
	}

	tab, err := LoadTable(cfg.Nodes)
	if err != nil {
		t.Fatal(err)
	}
	addr, err := tab.Lookup("room-1")
	if err != nil {
		t.Fatal(err)
	}
	if addr != "127.0.0.1:18080" && addr != "sfu2.example.com:18080" {
		t.Fatalf("unexpected signaling %q", addr)
	}
}

func TestLookupNodeUnknown(t *testing.T) {
	_, err := LookupNode(map[string]Node{"n1": {Signaling: "a:18080"}}, "n9")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadConfigMissingSignaling(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	content := `
[nodes.n1]
public_ip = "1.2.3.4"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected error")
	}
}
