package main

import (
	"sync"

	"github.com/pactus-project/pactus/committee"
	"github.com/pactus-project/pactus/crypto/hash"
	"github.com/pactus-project/pactus/sortition"
)

type Thread struct {
	body func(hash hash.Hash, cmt committee.Committee, seed sortition.VerifiableSeed, start, len int) []JoinedValKey
	mux  sync.Mutex
	vals []JoinedValKey
}

func NewThread(body func(hash hash.Hash, cmt committee.Committee, seed sortition.VerifiableSeed, start, len int) []JoinedValKey) *Thread {
	return &Thread{
		body: body,
	}
}

func (thread *Thread) Start(hash hash.Hash, cmt committee.Committee, seed sortition.VerifiableSeed, start, len int) {
	thread.mux.Lock()
	go thread.run(hash, cmt, seed, start, len)
}

func (thread *Thread) Join() []JoinedValKey {
	thread.mux.Lock()
	defer thread.mux.Unlock()

	return thread.vals
}

func (thread *Thread) run(hash hash.Hash, cmt committee.Committee, seed sortition.VerifiableSeed, start, len int) {
	thread.vals = thread.body(hash, cmt, seed, start, len)
	thread.mux.Unlock()
}
