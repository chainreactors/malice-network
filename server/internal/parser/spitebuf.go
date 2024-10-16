package parser

import (
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
)

func NewSpitesBuf() *SpitesCache {
	return &SpitesCache{cache: []*implantpb.Spite{}}
}

type SpitesCache struct {
	cache []*implantpb.Spite
	max   int
}

func (sc *SpitesCache) Len() int {
	return len(sc.cache)
}

func (sc *SpitesCache) Build() *implantpb.Spites {
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{}}
	for _, s := range sc.cache {
		spites.Spites = append(spites.Spites, s)
	}
	sc.Reset()
	return spites
}

func (sc *SpitesCache) BuildOrEmpty() *implantpb.Spites {
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{}}
	if len(sc.cache) == 0 {
		spites.Spites = append(spites.Spites, &implantpb.Spite{Body: &implantpb.Spite_Empty{}})
	} else {
		spites.Spites = append(spites.Spites, sc.cache...)
		spites.Reset()
	}
	return spites
}

func (sc *SpitesCache) Reset() {
	sc.cache = []*implantpb.Spite{}
}

func (sc *SpitesCache) Append(spite *implantpb.Spite) {
	sc.cache = append(sc.cache, spite)
}
