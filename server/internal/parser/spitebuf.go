package parser

import (
	"sync"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

func NewSpitesBuf() *SpitesCache {
	return &SpitesCache{cache: []*implantpb.Spite{}}
}

type SpitesCache struct {
	mu    sync.Mutex
	cache []*implantpb.Spite
	max   int
}

func (sc *SpitesCache) Len() int {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return len(sc.cache)
}

func (sc *SpitesCache) Build() *implantpb.Spites {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	spites := &implantpb.Spites{Spites: []*implantpb.Spite{}}
	for _, s := range sc.cache {
		spites.Spites = append(spites.Spites, s)
	}
	sc.cache = []*implantpb.Spite{}
	return spites
}

func (sc *SpitesCache) BuildOrEmpty() *implantpb.Spites {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	spites := &implantpb.Spites{Spites: []*implantpb.Spite{}}
	if len(sc.cache) == 0 {
		spites.Spites = append(spites.Spites, &implantpb.Spite{Body: &implantpb.Spite_Empty{}})
	} else {
		spites.Spites = append(spites.Spites, sc.cache...)
		sc.cache = []*implantpb.Spite{}
	}
	return spites
}

func (sc *SpitesCache) Reset() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.cache = []*implantpb.Spite{}
}

func (sc *SpitesCache) Append(spite *implantpb.Spite) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.cache = append(sc.cache, spite)
}
