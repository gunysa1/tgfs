package ipc

// Request is a command sent from CLI to daemon.
type Request struct {
	Command string            `json:"command"`
	Args    map[string]string `json:"args,omitempty"`
}

// Response is the daemon's reply.
type Response struct {
	OK    bool              `json:"ok"`
	Error string            `json:"error,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
	Lines []string          `json:"lines,omitempty"`
}
