package main

import (
	"fmt"
	"net"
	"os"
)

// chanFromConn creates a channel from a Conn object, and sends everything it
//  Read()s from the socket to the channel.
func chanFromConn(conn net.Conn) chan []byte {
	c := make(chan []byte)

	go func() {
		b := make([]byte, 1024)

		for {
			n, err := conn.Read(b)
			if n > 0 {
				res := make([]byte, n)
				// Copy the buffer so it doesn't get changed while read by the recipient.
				copy(res, b[:n])
				c <- res
			}
			if err != nil {
				c <- nil
				break
			}
		}
	}()

	return c
}

// Pipe creates a full-duplex pipe between the two sockets and transfers data from one to the other.
func Pipe(conn1 net.Conn, conn2 net.Conn) {
	chan1 := chanFromConn(conn1)
	chan2 := chanFromConn(conn2)

	for {
		select {
		case b1 := <-chan1:
			if b1 == nil {
				return
			} else {
				_, err := conn2.Write(b1)
				if err != nil {
					return
				}
			}
		case b2 := <-chan2:
			if b2 == nil {
				return
			} else {
				_, err := conn1.Write(b2)
				if err != nil {
					return
				}
			}
		}
	}
}

func forward(conn net.Conn, dstAddr string) {
    forwardConn, err := net.Dial("tcp", dstAddr)
    if err != nil {
    	fmt.Printf("[forward] client=(%v) dst=(%v) connect fail: %v", conn.RemoteAddr().String(), dstAddr, err)
        return
    }
    go Pipe(conn, forwardConn)
}

func printUsage(name string) {
	fmt.Printf("Usage:\n  %v LISTEN_ADDR REMOTE_ADDR\nEg:\n  %v 127.0.0.1:8080 192.168.0.1:80\n", name, name)
}

func main() {
	if len(os.Args) != 3 {
		printUsage(os.Args[0])
		os.Exit(1)
	}

	listenAddr := os.Args[1]
	remoteAddr := os.Args[2]

	ln, err := net.Listen("tcp", listenAddr)
    if err != nil {
		fmt.Printf("listen error: %v\n", err)
		os.Exit(1)
    }

	fmt.Printf("[started] %v -> %v", listenAddr, remoteAddr);

    for {
        conn, err := ln.Accept()
        if err != nil {
        	fmt.Printf("[accept] error: %v\n", err)
        	continue
        }

        fmt.Printf("[accept] from: %v", conn.RemoteAddr().String())
        go forward(conn, remoteAddr)
    }
}
