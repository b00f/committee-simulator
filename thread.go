package main

import (
	"sync"

	"github.com/pactus-project/pactus/committee"
	"github.com/pactus-project/pactus/sortition"
	"github.com/pactus-project/pactus/types/tx"
)

type Thread struct {
	body func(height uint32, cmt committee.Committee, seed sortition.VerifiableSeed, start, len int) []*tx.Tx
	mux  sync.Mutex
	vals []*tx.Tx
}

func NewThread(body func(height uint32, cmt committee.Committee, seed sortition.VerifiableSeed, start, len int) []*tx.Tx) *Thread {
	return &Thread{
		body: body,
	}
}

func (thread *Thread) Start(height uint32, cmt committee.Committee, seed sortition.VerifiableSeed, start, len int) {
	thread.mux.Lock()
	go thread.run(height, cmt, seed, start, len)
}

func (thread *Thread) Join() []*tx.Tx {
	thread.mux.Lock()
	defer thread.mux.Unlock()

	return thread.vals
}

func (thread *Thread) run(height uint32, cmt committee.Committee, seed sortition.VerifiableSeed, start, len int) {
	thread.vals = thread.body(height, cmt, seed, start, len)
	thread.mux.Unlock()
}
