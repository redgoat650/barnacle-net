package inflight

import (
	"math/rand"
	"sync"

	"github.com/redgoat650/barnacle-net/internal/message"
)

type Inflight struct {
	mu *sync.Mutex
	m  map[uint64]chan *message.Response
}

func NewInflight() *Inflight {
	return &Inflight{
		mu: new(sync.Mutex),
		m:  make(map[uint64]chan *message.Response),
	}
}

func (i *Inflight) Register() (uint64, chan *message.Response) {
	id := rand.Uint64()
	ch := make(chan *message.Response, 1)

	for {
		if _, ok := i.m[id]; !ok {
			i.m[id] = ch

			return id, ch
		}
		id = rand.Uint64()
	}
}

func (i *Inflight) Unregister(id uint64) {
	ch, ok := i.Get(id)
	if ok && ch != nil {
		close(ch)
	}
}

func (i *Inflight) Get(id uint64) (chan *message.Response, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()

	ch, ok := i.m[id]

	defer delete(i.m, id)

	return ch, ok
}
