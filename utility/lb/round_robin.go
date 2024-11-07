package lb

import "sync"

type RoundRobin struct {
	currentIndex int
	mutex        sync.Mutex
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

func (r *RoundRobin) Index(lens int) (index int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.currentIndex >= lens {
		r.currentIndex = 0
	}

	index = r.currentIndex

	r.currentIndex = (r.currentIndex + 1) % lens

	return
}

func (r *RoundRobin) Pick(values []string) string {
	return values[r.Index(len(values))]
}
