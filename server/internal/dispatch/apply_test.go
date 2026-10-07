package dispatch

import "testing"

func TestServerParamsFromNode(t *testing.T) {
	p, err := ServerParamsFromNode(Node{
		Signaling: "203.0.113.10:18080",
		PublicIP:  "203.0.113.10",
		UDPPort:   18081,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Port != 18080 || p.PublicIP != "203.0.113.10" || p.UDPPort != 18081 {
		t.Fatalf("%+v", p)
	}
}

func TestServerParamsFromNodeBadSignaling(t *testing.T) {
	if _, err := ServerParamsFromNode(Node{Signaling: "no-port"}); err == nil {
		t.Fatal("expected error")
	}
}
