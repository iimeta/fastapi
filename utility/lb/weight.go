package lb

import (
	"sync"
)

type Weight struct {
	Servers []Server
	mutex   sync.Mutex
}
type Server struct {
	Name           string
	OriginalWeight int
	CurrentWeight  int
}

func NewWeight(servers []Server) *Weight {
	return &Weight{
		Servers: servers,
	}
}

func (w *Weight) Pick() *Server {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if len(w.Servers) == 0 {
		return nil
	}

	if len(w.Servers) == 1 {
		return &w.Servers[0]
	}

	totalWeight := 0
	selected := &w.Servers[0]

	for i := range w.Servers {

		server := &w.Servers[i]
		totalWeight += server.OriginalWeight
		server.CurrentWeight += server.OriginalWeight

		if server.CurrentWeight > selected.CurrentWeight {
			selected = server
		}
	}

	selected.CurrentWeight -= totalWeight

	return selected
}
