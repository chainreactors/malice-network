package types

import "github.com/chainreactors/malice-network/proto/implant/commonpb"

func NewSpitesCache() *SpitesCache {
	return &SpitesCache{cache: []*commonpb.Spite{}}
}

type SpitesCache struct {
	cache []*commonpb.Spite
	max   int
}

func (sc *SpitesCache) Len() int {
	return len(sc.cache)
}

func (sc *SpitesCache) Build() *commonpb.Spites {
	spites := &commonpb.Spites{Spites: []*commonpb.Spite{}}
	for _, s := range sc.cache {
		spites.Spites = append(spites.Spites, s)
	}
	sc.Reset()
	return spites
}

func (sc *SpitesCache) BuildOrEmpty() *commonpb.Spites {
	spites := &commonpb.Spites{Spites: []*commonpb.Spite{}}
	if len(sc.cache) == 0 {
		spites.Spites = append(spites.Spites, &commonpb.Spite{Body: &commonpb.Spite_Empty{}})
	} else {
		spites.Spites = append(spites.Spites, sc.cache...)
		spites.Reset()
	}
	return spites
}

func (sc *SpitesCache) Reset() {
	sc.cache = []*commonpb.Spite{}
}

func (sc *SpitesCache) Append(spite *commonpb.Spite) {
	sc.cache = append(sc.cache, spite)
}
