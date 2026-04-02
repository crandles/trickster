package dataset

import (
	"sync"

	"github.com/trickstercache/trickster/v2/pkg/util/sets"
)

const maxSeriesMapSize = 10000

var seriesHashMapPool = sync.Pool{
	New: func() any {
		return make(map[Hash]*Series, 64)
	},
}

func getSeriesHashMap() map[Hash]*Series {
	m := seriesHashMapPool.Get().(map[Hash]*Series)
	clear(m)
	return m
}

func putSeriesHashMap(m map[Hash]*Series) {
	if m == nil || len(m) > maxSeriesMapSize {
		return
	}
	seriesHashMapPool.Put(m)
}

var hashSetPool = sync.Pool{
	New: func() any {
		return make(sets.Set[Hash], 64)
	},
}

func getHashSet() sets.Set[Hash] {
	s := hashSetPool.Get().(sets.Set[Hash])
	clear(s)
	return s
}

func putHashSet(s sets.Set[Hash]) {
	if s == nil || len(s) > maxSeriesMapSize {
		return
	}
	hashSetPool.Put(s)
}
