package lb

import (
	"github.com/iimeta/fastapi/v2/internal/model"
)

type Weight struct {
	ModelAgents []*model.ModelAgent
	Keys        []*model.Key
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

func (w *Weight) PickModelAgent() *model.ModelAgent {

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
