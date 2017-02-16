// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
)

type Client struct {
	c net.Conn

	gr *gob.Encoder
	gw *gob.Decoder
}

func NewClient(socketPath string) *Client {
	c, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatal(err)
	}

	// Send credentials.
	err = writeCredentials(c.(*net.UnixConn))
	if err != nil {
		log.Fatal("failed to send credentials: ", err)
	}

	gr, gw := gob.NewEncoder(c), gob.NewDecoder(c)

	return &Client{c, gr, gw}
}

func (c *Client) do(action PerfLockAction, response interface{}) {
	err := c.gr.Encode(action)
	if err != nil {
		log.Fatal(err)
	}

	err = c.gw.Decode(response)
	if err != nil {
		log.Fatal(err)
	}
}

func (c *Client) Acquire(shared, nonblocking bool, msg string) bool {
	var ok bool
	c.do(PerfLockAction{ActionAcquire{Shared: shared, NonBlocking: nonblocking, Msg: msg}}, &ok)
	return ok
}

func (c *Client) List() []string {
	var list []string
	c.do(PerfLockAction{ActionList{}}, &list)
	return list
}

func (c *Client) SetGovernor(percent int) error {
	var err string
	c.do(PerfLockAction{ActionSetGovernor{Percent: percent}}, &err)
	if err == "" {
		return nil
	}
	return fmt.Errorf("%s", err)
}
