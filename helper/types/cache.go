package types

import "github.com/chainreactors/malice-network/proto/implant/commonpb"

type SpitesCache struct {
	cache []*commonpb.Spite
	max   int
}

func (sc SpitesCache) Len() int {
	return len(sc.cache)
}

func (sc SpitesCache) Build() *commonpb.Spites {
	spites := &commonpb.Spites{Spites: []*commonpb.Spite{}}
	for _, s := range sc.cache {
		spites.Spites = append(spites.Spites, s)
	}
	spites.Reset()
	return spites
}

func (sc SpitesCache) Reset() {
	sc.cache = []*commonpb.Spite{}
}

func (sc SpitesCache) Append(spite *commonpb.Spite) {
	sc.cache = append(sc.cache, spite)
}
