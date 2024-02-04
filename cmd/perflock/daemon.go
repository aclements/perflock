// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"runtime"
	"time"

	"github.com/aclements/perflock/internal/cpupower"
)

var theLock PerfLock

func doDaemon(path string) {
	// TODO: Don't start if another daemon is already running.

	// Linux supports an abstract namespace for UNIX domain sockets (see unix(7)).
	// These do not involve the filesystem, and are world-connectable.
	isAbstractSocket := runtime.GOOS == "linux" && len(path) > 1 && path[0] == '@'
	if !isAbstractSocket {
		os.Remove(path)
	}
	l, err := net.Listen("unix", path)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	if !isAbstractSocket {
		err = os.Chmod(path, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Receive connections.
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func(c net.Conn) {
			defer c.Close()
			NewServer(c).Serve()
		}(conn)
	}
}

type Server struct {
	c        net.Conn
	userName string

	locker    *Locker
	acquiring bool

	oldGovernors []*governorSettings
}

func NewServer(c net.Conn) *Server {
	return &Server{c: c}
}

func (s *Server) Serve() {
	// Drop any held locks if we exit for any reason.
	defer s.drop()

	// Get connection credentials.
	ucred, err := readCredentials(s.c.(*net.UnixConn))
	if err != nil {
		log.Print("reading credentials: ", err)
		return
	}

	u, err := user.LookupId(fmt.Sprintf("%d", ucred.Uid))
	s.userName = "???"
	if err == nil {
		s.userName = u.Username
	}

	// Receive incoming actions. We do this in a goroutine so the
	// main handler can select on EOF or lock acquisition.
	actions := make(chan PerfLockAction)
	go func() {
		gr := gob.NewDecoder(s.c)
		for {
			var msg PerfLockAction
			err := gr.Decode(&msg)
			if err != nil {
				if err != io.EOF {
					log.Print(err)
				}
				close(actions)
				return
			}
			actions <- msg
		}
	}()

	// Process incoming actions.
	var acquireC <-chan bool
	gw := gob.NewEncoder(s.c)
	for {
		select {
		case action, ok := <-actions:
			if !ok {
				// Connection closed.
				return
			}
			if s.acquiring {
				log.Printf("protocol error: message while acquiring")
				return
			}
			switch action := action.Action.(type) {
			case ActionAcquire:
				if s.locker != nil {
					log.Printf("protocol error: acquiring lock twice")
					return
				}
				msg := fmt.Sprintf("%s\t%s\t%s", s.userName, time.Now().Format(time.Stamp), action.Msg)
				if action.Shared {
					msg += " [shared]"
				}
				s.locker = theLock.Enqueue(action.Shared, action.NonBlocking, msg)
				if s.locker != nil {
					// Enqueued. Wait for acquire.
					s.acquiring = true
					acquireC = s.locker.C
				} else {
					// Non-blocking acquire failed.
					if err := gw.Encode(false); err != nil {
						log.Print(err)
						return
					}
				}

			case ActionList:
				list := theLock.Queue()
				if err := gw.Encode(list); err != nil {
					log.Print(err)
					return
				}

			case ActionSetGovernor:
				if s.locker == nil {
					log.Printf("protocol error: setting governor without lock")
					return
				}
				err := s.setGovernor(action.Percent)
				errString := ""
				if err != nil {
					errString = err.Error()
				}
				if err := gw.Encode(errString); err != nil {
					log.Print(err)
					return
				}

			default:
				log.Printf("unknown message")
				return
			}

		case <-acquireC:
			// Lock acquired.
			s.acquiring, acquireC = false, nil
			if err := gw.Encode(true); err != nil {
				log.Print(err)
				return
			}
		}
	}
}

func (s *Server) drop() {
	// Restore the CPU governor before releasing the lock.
	if s.oldGovernors != nil {
		s.restoreGovernor()
		s.oldGovernors = nil
	}
	// Release the lock.
	if s.locker != nil {
		theLock.Dequeue(s.locker)
		s.locker = nil
	}
}

type governorSettings struct {
	domain   *cpupower.Domain
	min, max int
}

func (s *Server) setGovernor(percent int) error {
	domains, err := cpupower.Domains()
	if err != nil {
		return err
	}
	if len(domains) == 0 {
		return fmt.Errorf("no power domains")
	}

	// Save current frequency settings.
	old := []*governorSettings{}
	for _, d := range domains {
		min, max, err := d.CurrentRange()
		if err != nil {
			return err
		}
		old = append(old, &governorSettings{d, min, max})
	}
	s.oldGovernors = old

	// Set new settings.
	abs := func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	}
	for _, d := range domains {
		min, max, avail := d.AvailableRange()
		target := (max-min)*percent/100 + min

		// Find the nearest available frequency.
		if len(avail) != 0 {
			closest := avail[0]
			for _, a := range avail {
				if abs(target-a) < abs(target-closest) {
					closest = a
				}
			}
			target = closest
		}

		err := d.SetRange(target, target)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) restoreGovernor() error {
	var err error
	for _, g := range s.oldGovernors {
		// Try to set all of the domains, even if one fails.
		err1 := g.domain.SetRange(g.min, g.max)
		if err1 != nil && err == nil {
			err = err1
		}
	}
	return err
}
