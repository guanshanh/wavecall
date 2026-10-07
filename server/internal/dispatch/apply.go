package dispatch

import (
	"fmt"
	"net"
	"strconv"
)

// ServerParams are SFU listen / ICE fields filled from a cluster node entry.
type ServerParams struct {
	Port     int
	PublicIP string
	UDPPort  int
}

// ServerParamsFromNode maps a table entry onto SFU process settings.
// Signaling's port becomes the HTTP/WebSocket listen port.
func ServerParamsFromNode(n Node) (ServerParams, error) {
	_, portStr, err := net.SplitHostPort(n.Signaling)
	if err != nil {
		return ServerParams{}, fmt.Errorf("dispatch: signaling %q: %w", n.Signaling, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return ServerParams{}, fmt.Errorf("dispatch: signaling port %q: %w", portStr, err)
	}
	if port <= 0 || port > 65535 {
		return ServerParams{}, fmt.Errorf("dispatch: invalid signaling port %d", port)
	}
	return ServerParams{
		Port:     port,
		PublicIP: n.PublicIP,
		UDPPort:  n.UDPPort,
	}, nil
}
