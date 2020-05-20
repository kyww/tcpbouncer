package main

import (
    "fmt"
    "net"
    "os"
    "log"
    "bytes"
)

var logger = log.New(os.Stdout, "", log.Ldate | log.Ltime)
var secret string
var ipSet = make(map[string]bool)

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
    logger.Printf("[forward_start] client=(%v) dst=(%v)", conn.RemoteAddr().String(), dstAddr)
    forwardConn, err := net.Dial("tcp", dstAddr)
    if err != nil {
        logger.Printf("[forward_end] client=(%v) dst=(%v) connect fail: %v", conn.RemoteAddr().String(), dstAddr, err)
        return
    }
    Pipe(conn, forwardConn)
    logger.Printf("[forward_end] client=(%v) dst=(%v)", conn.RemoteAddr().String(), dstAddr)
}

func printUsage(name string) {
    fmt.Printf("Usage:\n  %v LISTEN_ADDR REMOTE_ADDR [SECRET]\nEg:\n  %v 127.0.0.1:8080 192.168.0.1:80\n  %v 127.0.0.1:8080 192.168.0.1:80 mysecret\n", name, name, name)
}

func auth(conn net.Conn, ip string) {
    b := make([]byte, 1024)
    n, err := conn.Read(b)
    if n > 0 && err == nil && -1 != bytes.Index(b, []byte(secret)) {
        ipSet[ip] = true
        logger.Printf("[auth_success] from: %v, added ip: %v", conn.RemoteAddr().String(), ip)
    } else {
        logger.Printf("[auth_fail] from: %v", conn.RemoteAddr().String())
    }

    conn.Close()
}

func main() {
        argCount := len(os.Args)
    if argCount != 3 && argCount != 4 {
        printUsage(os.Args[0])
        os.Exit(1)
    }

    listenAddr := os.Args[1]
    dstAddr := os.Args[2]
        if (argCount == 4) {
                secret = os.Args[3]
        }

    ln, err := net.Listen("tcp", listenAddr)
    if err != nil {
        logger.Printf("listen error: %v\n", err)
        os.Exit(1)
    }

    logger.Printf("[started] %v -> %v, secret=%v", listenAddr, dstAddr, secret);

    for {
        conn, err := ln.Accept()
        if err != nil {
            logger.Printf("[accept] error: %v\n", err)
            continue
        }

        logger.Printf("[accept] from: %v\n", conn.RemoteAddr().String())
        ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()

        if secret != "" && ipSet[ip] != true {
            go auth(conn, ip)
        } else {
            go forward(conn, dstAddr)
        }
    }
}
