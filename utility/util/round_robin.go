package util

import "sync"

type RoundRobin struct {
	mu       sync.Mutex
	CurIndex int
}

func (r *RoundRobin) Index(lens int) (index int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.CurIndex >= lens {
		r.CurIndex = 0
	}

	index = r.CurIndex

	r.CurIndex = (r.CurIndex + 1) % lens

	return
}

func (r *RoundRobin) PickKey(keys []string) string {
	return keys[r.Index(len(keys))]
}
