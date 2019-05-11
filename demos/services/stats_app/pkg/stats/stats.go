package stats

import (
	"sync"
	"time"
)

type Stats struct {
	lock                 sync.Mutex
	conversations        map[string]*Conversation
	startTime            int64
	totalInboundRequests int
	totalRequestErrors   int
}

type Conversation struct {
	Neighbor string
	Requests int
}

func NewStats() *Stats {
	conversations := map[string]*Conversation{}
	return &Stats{
		lock:          sync.Mutex{},
		startTime:     time.Now().Unix(),
		conversations: conversations,
	}
}

func NewConversation(name string) *Conversation {
	return &Conversation{
		Neighbor: name,
		Requests: 0,
	}
}

func (s *Stats) IncrementConversation(name string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.totalInboundRequests++
	if _, ok := s.conversations[name]; ok {
		s.conversations[name].Requests++
	} else {
		s.conversations[name] = NewConversation(name)
	}
}

func (s *Stats) IncrementErrors() {
	s.totalRequestErrors++
}

func (s *Stats) TotalOutboundRequests() int {
	return s.totalInboundRequests
}

func (s *Stats) TotalOutboundRequestErrors() int {
	return s.totalRequestErrors
}

func (s *Stats) RecordDelta(delta int) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.totalInboundRequests++

}
