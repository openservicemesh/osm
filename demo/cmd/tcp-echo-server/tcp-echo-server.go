// package main implements a TCP echo server that echoes back the TCP client's request as a part of its response.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log            = logger.NewPretty("tcp-echo-server")
	port           = flag.Int("port", 9090, "port on which this app is serving TCP connections")
	responsePrefix = "echo response:"
)

func main() {
	flag.Parse()

	listenAddr := fmt.Sprintf(":%d", *port)

	// Create a tcp listener
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating TCP listener on address %q", listenAddr)
	}
	log.Info().Msgf("Server listening on address %q", listenAddr)

	// listen for new connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error().Err(err).Msg("Error accepting connection")
			continue
		}

		go echoResponse(conn)
	}
}

func echoResponse(conn net.Conn) {
	defer conn.Close() //nolint: errcheck,gosec
	reader := bufio.NewReader(conn)

	for {
		requestMsg, err := reader.ReadString(byte('\n'))
		requestMsg = strings.TrimSuffix(requestMsg, "\n") // trim trailing newline character

		switch err {
		case nil:
			// respond to the request, prepend the prefix to the response
			fmt.Printf("Received request: [%s]\n", requestMsg)
			response := fmt.Sprintf("%s %s\n", responsePrefix, requestMsg)
			if bytesWritten, writeErr := conn.Write([]byte(response)); err != nil {
				log.Error().Err(writeErr).Msg("Write error")
			} else {
				log.Debug().Msgf("Wrote %d bytes", bytesWritten)
			}
			fmt.Printf("Response sent: [%s]\n", response)

		case io.EOF:
			return

		default:
			// unexpected error
			fmt.Printf("Unexpected error: %s\n", err)
			return
		}
	}
}
