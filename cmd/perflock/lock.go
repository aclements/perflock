// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "sync"

type PerfLock struct {
	l sync.Mutex
	q []*Locker
}

type Locker struct {
	C      <-chan bool
	c      chan<- bool
	shared bool
	woken  bool

	msg string
}

func (l *PerfLock) Enqueue(shared, nonblocking bool, msg string) *Locker {
	ch := make(chan bool, 1)
	locker := &Locker{ch, ch, shared, false, msg}

	// Enqueue.
	l.l.Lock()
	defer l.l.Unlock()
	l.setQ(append(l.q, locker))

	if nonblocking && !locker.woken {
		// Acquire failed. Dequeue.
		l.setQ(l.q[:len(l.q)-1])
		return nil
	}

	return locker
}

func (l *PerfLock) Dequeue(locker *Locker) {
	l.l.Lock()
	defer l.l.Unlock()
	for i, o := range l.q {
		if locker == o {
			copy(l.q[i:], l.q[i+1:])
			l.setQ(l.q[:len(l.q)-1])
			return
		}
	}
	panic("Dequeue of non-enqueued Locker")
}

func (l *PerfLock) Queue() []string {
	var q []string

	l.l.Lock()
	defer l.l.Unlock()
	for _, locker := range l.q {
		q = append(q, locker.msg)
	}
	return q
}

func (l *PerfLock) setQ(q []*Locker) {
	l.q = q
	if len(q) == 0 {
		return
	}

	wake := func(locker *Locker) {
		if locker.woken == false {
			locker.woken = true
			locker.c <- true
		}
	}
	if q[0].shared {
		// Wake all shared acquires at the head of the queue.
		for _, locker := range q {
			if !locker.shared {
				break
			}
			wake(locker)
		}
	} else {
		wake(q[0])
	}
}
