package main

import (
    "bufio"
    "errors"
    "flag"
    "fmt"
    "io"
    "net"
    "os"
    "strconv"
    "strings"
    "sync"
)

func handleConnection(incomingConn net.Conn) error {
    defer incomingConn.Close()
    fmt.Println("Got a connection from", incomingConn.RemoteAddr())

    bufreader := bufio.NewReader(incomingConn)
    header, err := bufreader.ReadString('\n')
    if err != nil {
        return err
    }
    header = strings.Trim(header, " \n\r")

    components := strings.Split(header, " ")
    if len(components) != 6 {
        return errors.New("Invalid PROXY line")
    }

    // The address family must be on the 2nd field.
    family := components[1]
    // The destination address must be on the 4th field.
    dstAddr := components[3]
    // The destination port must be on the 6th field.
    dstPort := components[5]

    switch family {
        case "TCP4":
        case "TCP6":
            dstAddr = "[" + dstAddr + "]"
        default:
            return errors.New("Unknown address family")
    }

    dstAddrPort := dstAddr + ":" + dstPort
    outgoingConn, err := net.Dial("tcp", dstAddrPort)
    if err != nil {
        return err
    }

    fmt.Println("Forwarding from", incomingConn.RemoteAddr(), "to", outgoingConn.RemoteAddr())

    // Forwarding
    var wg sync.WaitGroup
    wg.Add(2)
    go func () {
        defer wg.Done()
        _, err = io.Copy(incomingConn, outgoingConn)
    }()
    go func () {
        defer wg.Done()
        _, err = io.Copy(outgoingConn, incomingConn)
    }()
    wg.Wait()

    return err
}

func main() {
    // List of available options.
    port := flag.Int("p", 8080, "The port for the proxy server to listen")
    flag.Parse()

    listener, err := net.Listen("tcp", ":" + strconv.Itoa(*port))
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        return
    }
    fmt.Println("Listening on", listener.Addr())
    defer listener.Close()

    for {
        // Listen for the incoming connection.
        conn, err := listener.Accept()
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            break
        }
        go func () {
            err = handleConnection(conn)
            if err != nil {
                fmt.Fprintln(os.Stderr, "connection from " + conn.RemoteAddr().String() + ":", err)
            }
        }()
    }
}
