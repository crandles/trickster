/*
 * Copyright 2018 The Trickster Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
