package config

// Config holds all server configuration.
type Config struct {
	Port       int    // HTTP/WebSocket listen port
	PublicIP   string // server public IP; advertised to clients directly (required on cloud VMs with 1:1 NAT)
	UDPPort    int    // single UDP port for WebRTC media (UDPMux); 0 = ephemeral ports
	STUNServer string // optional STUN server fallback; empty = none
	TURNServer string // TURN server address for ICE relay
	TURNUser   string // TURN username
	TURNPass   string // TURN password
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		Port:       18080,
		UDPPort:    18081,
		STUNServer: "",
		TURNServer: "",
		TURNUser:   "",
		TURNPass:   "",
	}
}

// ICEServers returns the ICE server configuration for WebRTC PeerConnections.
func (c *Config) ICEServers() []ICEServer {
	var servers []ICEServer
	if c.STUNServer != "" {
		servers = append(servers, ICEServer{URLs: []string{c.STUNServer}})
	}
	if c.TURNServer != "" {
		servers = append(servers, ICEServer{
			URLs:       []string{c.TURNServer},
			Username:   c.TURNUser,
			Credential: c.TURNPass,
		})
	}
	return servers
}

// ICEServer represents a single ICE server entry sent to clients.
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}
