package dispatch

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

var ErrEmptyTable = errors.New("dispatch: empty node table")

type Table struct {
	nodes []string // signaling host:port, sorted by config key
}

func LoadTable(nodes map[string]Node) (*Table, error) {
	if len(nodes) == 0 {
		return nil, ErrEmptyTable
	}
	keys := make([]string, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		addr := strings.TrimSpace(nodes[k].Signaling)
		if addr == "" {
			return nil, fmt.Errorf("dispatch: empty signaling for node %q", k)
		}
		out = append(out, addr)
	}
	return &Table{nodes: out}, nil
}

func (t *Table) Lookup(roomID string) (string, error) {
	if t == nil || len(t.nodes) == 0 {
		return "", ErrEmptyTable
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(roomID))
	idx := h.Sum64() % uint64(len(t.nodes))
	return t.nodes[idx], nil
}
