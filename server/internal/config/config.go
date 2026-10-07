package config

// Config holds all server configuration.
type Config struct {
	Port       int    // HTTP/WebSocket listen port
	STUNServer string // STUN server address for ICE
	TURNServer string // TURN server address for ICE relay
	TURNUser   string // TURN username
	TURNPass   string // TURN password
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		Port:       8080,
		STUNServer: "stun:stun.l.google.com:19302",
		TURNServer: "",
		TURNUser:   "",
		TURNPass:   "",
	}
}

// ICEServers returns the ICE server configuration for WebRTC PeerConnections.
func (c *Config) ICEServers() []ICEServer {
	servers := []ICEServer{
		{URLs: []string{c.STUNServer}},
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
