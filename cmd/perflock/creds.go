// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"os"
	"syscall"
)

// TODO: Use SO_PEERCRED instead?

func writeCredentials(c *net.UnixConn) error {
	ucred := syscall.Ucred{Pid: int32(os.Getpid()), Uid: uint32(os.Getuid()), Gid: uint32(os.Getgid())}
	credOob := syscall.UnixCredentials(&ucred)
	credMsg := []byte("x")
	n, oobn, err := c.WriteMsgUnix(credMsg, credOob, nil)
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("short send (%d bytes)", n)
	}
	if oobn != len(credOob) {
		return fmt.Errorf("short OOB send (%d bytes)", oobn)
	}
	return nil
}

func readCredentials(c *net.UnixConn) (*syscall.Ucred, error) {
	// Enable receiving credentials on c.
	f, err := c.File()
	if err != nil {
		return nil, err
	}
	err = syscall.SetsockoptInt(int(f.Fd()), syscall.SOL_SOCKET, syscall.SO_PASSCRED, 1)
	f.Close()
	if err != nil {
		return nil, err
	}

	// Receive credentials.
	buf := make([]byte, 1)
	oob := make([]byte, 128)
	n, oobn, _, _, err := c.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, fmt.Errorf("expected 1 byte, got %d", n)
	}

	// Parse OOB data.
	scms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return nil, err
	}
	if len(scms) != 1 {
		return nil, fmt.Errorf("expected 1 control message, got %d", len(scms))
	}
	return syscall.ParseUnixCredentials(&scms[0])
}
