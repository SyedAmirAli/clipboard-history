//go:build windows

package ipc

import (
	"fmt"
	"net"
	"time"
)

// Listener opens a named pipe for IPC on Windows.
type Listener struct {
	pipePath string
	listener net.Listener
	done     chan struct{}
}

// Listen creates a named pipe listener for Windows IPC.
func Listen(pipePath string, handler func(string)) (*Listener, error) {
	// On Windows, use net.Listen with "pipe" network
	listener, err := net.Listen("pipe", pipePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create named pipe listener: %w", err)
	}

	ln := &Listener{
		pipePath: pipePath,
		listener: listener,
		done:     make(chan struct{}),
	}

	go ln.acceptLoop(handler)
	return ln, nil
}

func (ln *Listener) acceptLoop(handler func(string)) {
	defer close(ln.done)
	for {
		conn, err := ln.listener.Accept()
		if err != nil {
			// Listener was closed
			return
		}

		go func(c net.Conn) {
			defer c.Close()
			c.SetReadDeadline(time.Now().Add(2 * time.Second))

			buf := make([]byte, 256)
			n, err := c.Read(buf)
			if err != nil {
				return
			}

			cmd := string(buf[:n])
			handler(cmd)
		}(conn)
	}
}

// Close closes the named pipe listener.
func (ln *Listener) Close() error {
	if ln.listener != nil {
		ln.listener.Close()
		<-ln.done
	}
	return nil
}

// Send sends a command to the running instance via named pipe.
func Send(pipePath string, cmd string) error {
	conn, err := net.Dial("pipe", pipePath)
	if err != nil {
		return fmt.Errorf("failed to connect to named pipe: %w", err)
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	return nil
}

// IsRunning checks if a named pipe listener is active.
func IsRunning(pipePath string) bool {
	conn, err := net.Dial("pipe", pipePath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
