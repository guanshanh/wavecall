package dispatch

import (
	"testing"
)

func TestLookupEmpty(t *testing.T) {
	_, err := LoadTable(nil)
	if err != ErrEmptyTable {
		t.Fatalf("got %v, want ErrEmptyTable", err)
	}
}

func TestLoadTableBlankNode(t *testing.T) {
	_, err := LoadTable(map[string]Node{"n1": {Signaling: "  "}})
	if err == nil {
		t.Fatal("expected error for blank signaling")
	}
}

func TestLookupStableAndOrdered(t *testing.T) {
	// Keys out of alpha order in the map literal; Lookup must sort keys.
	tab, err := LoadTable(map[string]Node{
		"n2": {Signaling: "b.example:18080"},
		"n1": {Signaling: "a.example:18080"},
		"n3": {Signaling: "c.example:18080"},
	})
	if err != nil {
		t.Fatal(err)
	}
	n1, err := tab.Lookup("room-42")
	if err != nil {
		t.Fatal(err)
	}
	n2, err := tab.Lookup("room-42")
	if err != nil {
		t.Fatal(err)
	}
	if n1 != n2 {
		t.Fatalf("unstable: %q vs %q", n1, n2)
	}
	allowed := map[string]bool{
		"a.example:18080": true,
		"b.example:18080": true,
		"c.example:18080": true,
	}
	if !allowed[n1] {
		t.Fatalf("unexpected node %q", n1)
	}
}

func TestLookupSingleNode(t *testing.T) {
	tab, err := LoadTable(map[string]Node{"n1": {Signaling: "only.example:18080"}})
	if err != nil {
		t.Fatal(err)
	}
	n, err := tab.Lookup("anything")
	if err != nil || n != "only.example:18080" {
		t.Fatalf("got %q %v", n, err)
	}
}
