package lb

import (
	"github.com/iimeta/fastapi/internal/model"
	"sync"
)

type Weight struct {
	Servers     []Server
	ModelAgents []*model.ModelAgent
	Keys        []*model.Key
	mutex       sync.Mutex
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

func NewModelAgentWeight(modelAgents []*model.ModelAgent) *Weight {
	return &Weight{
		ModelAgents: modelAgents,
	}
}

func NewKeyWeight(keys []*model.Key) *Weight {
	return &Weight{
		Keys: keys,
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

func (w *Weight) PickModelAgent() *model.ModelAgent {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if len(w.ModelAgents) == 0 {
		return nil
	}

	if len(w.ModelAgents) == 1 {
		return w.ModelAgents[0]
	}

	totalWeight := 0
	selected := w.ModelAgents[0]

	for i := range w.ModelAgents {

		modelAgent := w.ModelAgents[i]
		totalWeight += modelAgent.Weight
		modelAgent.CurrentWeight += modelAgent.Weight

		if modelAgent.CurrentWeight > selected.CurrentWeight {
			selected = modelAgent
		}
	}

	selected.CurrentWeight -= totalWeight

	return selected
}

func (w *Weight) PickKey() *model.Key {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if len(w.Keys) == 0 {
		return nil
	}

	if len(w.Keys) == 1 {
		return w.Keys[0]
	}

	totalWeight := 0
	selected := w.Keys[0]

	for i := range w.Keys {

		key := w.Keys[i]
		totalWeight += key.Weight
		key.CurrentWeight += key.Weight

		if key.CurrentWeight > selected.CurrentWeight {
			selected = key
		}
	}

	selected.CurrentWeight -= totalWeight

	return selected
}
