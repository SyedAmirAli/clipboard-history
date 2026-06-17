//go:build linux

package ipc

import (
	"bufio"
	"errors"
	"net"
	"os"
	"strings"
	"time"
)

// IsRunning reports whether a clipd instance is listening on the Unix socket.
func IsRunning(path string) bool {
	conn, err := net.DialTimeout("unix", path, 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Send delivers a command to the running instance via Unix socket.
func Send(path, cmd string) error {
	conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		return err
	}
	return nil
}

// Listener wraps the Unix socket listener for IPC.
type Listener struct {
	ln   net.Listener
	path string
}

// Listen starts accepting control connections on a Unix socket at path.
func Listen(path string, handler func(cmd string)) (*Listener, error) {
	// Clear stale socket
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Not fatal — net.Listen will surface a real conflict
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	l := &Listener{ln: ln, path: path}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				line, err := bufio.NewReader(c).ReadString('\n')
				if err != nil && line == "" {
					return
				}
				cmd := strings.TrimSpace(line)
				if ValidCommand(cmd) {
					handler(cmd)
				}
			}(conn)
		}
	}()

	return l, nil
}

// Close stops the Unix socket listener and removes the socket file.
func (l *Listener) Close() {
	if l == nil {
		return
	}
	if l.ln != nil {
		_ = l.ln.Close()
	}
	_ = os.Remove(l.path)
}
