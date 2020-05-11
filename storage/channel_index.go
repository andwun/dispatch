package storage

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type ChannelListItem struct {
	Name      string
	UserCount int
	Topic     string
}

type ChannelListIndex interface {
	Add(item *ChannelListItem)
	Finish()
	Search(q string) []*ChannelListItem
	SearchN(q string, start, n int) []*ChannelListItem
	len() int
}

type MapChannelListIndex struct {
	channels chanList
	m        map[string][]*ChannelListItem
}

func NewMapChannelListIndex() *MapChannelListIndex {
	return &MapChannelListIndex{
		m: map[string][]*ChannelListItem{},
	}
}

func (idx *MapChannelListIndex) Add(item *ChannelListItem) {
	idx.channels = append(idx.channels, item)
}

func (idx *MapChannelListIndex) Finish() {
	sort.Sort(idx.channels)

	for _, ch := range idx.channels {
		key := strings.TrimLeft(strings.ToLower(ch.Name), "#")

		for i := 1; i <= len(key); i++ {
			k := key[:i]
			if _, ok := idx.m[k]; ok {
				idx.m[k] = append(idx.m[k], ch)
			} else {
				idx.m[k] = chanList{ch}
			}
		}
	}
}

func (idx *MapChannelListIndex) Search(q string) []*ChannelListItem {
	if q == "" {
		return idx.channels
	}
	return idx.m[q]
}

func (idx *MapChannelListIndex) SearchN(q string, start, n int) []*ChannelListItem {
	if q == "" {
		if start >= len(idx.channels) {
			return nil
		}
		return idx.channels[start:min(start+n, len(idx.channels))]
	}

	res := idx.m[q]
	if start >= len(res) {
		return nil
	}
	return res[start:min(start+n, len(res))]
}

func (idx *MapChannelListIndex) len() int {
	return len(idx.channels)
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type chanList []*ChannelListItem

func (c chanList) Len() int {
	return len(c)
}

func (c chanList) Less(i, j int) bool {
	return c[i].UserCount > c[j].UserCount ||
		(c[i].UserCount == c[j].UserCount &&
			strings.ToLower(c[i].Name) < strings.ToLower(c[j].Name))
}

func (c chanList) Swap(i, j int) {
	ch := c[i]
	c[i] = c[j]
	c[j] = ch
}

const ChannelListUpdateInterval = time.Hour * 24
const ChannelListUpdateTimeout = time.Minute * 5

type ChannelIndexManager struct {
	indexes map[string]*managedChannelIndex
	lock    sync.Mutex
}

func NewChannelIndexManager() *ChannelIndexManager {
	return &ChannelIndexManager{
		indexes: map[string]*managedChannelIndex{},
	}
}

type managedChannelIndex struct {
	index     ChannelListIndex
	updatedAt time.Time
	updating  bool
}

func (m *ChannelIndexManager) Get(server string) (ChannelListIndex, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	idx, ok := m.indexes[server]
	if !ok {
		m.indexes[server] = &managedChannelIndex{
			updating: true,
		}
		go m.timeoutUpdate(server)
		return nil, true
	}

	if !idx.updating && time.Since(idx.updatedAt) > ChannelListUpdateInterval {
		idx.updating = true
		go m.timeoutUpdate(server)
		return idx.index, true
	}

	return idx.index, false
}

func (m *ChannelIndexManager) Set(server string, index ChannelListIndex) {
	if index.len() > 0 {
		m.lock.Lock()
		m.indexes[server] = &managedChannelIndex{
			index:     index,
			updatedAt: time.Now(),
		}
		m.lock.Unlock()
	}
}

func (m *ChannelIndexManager) timeoutUpdate(server string) {
	time.Sleep(ChannelListUpdateTimeout)

	m.lock.Lock()
	if m.indexes[server].updating {
		m.indexes[server].updating = false
	}
	m.lock.Unlock()
}
