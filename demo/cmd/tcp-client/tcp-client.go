// package main implements a TCP client that sends TCP data to a TCP echo server and prints the response.
// The clients does the following:
// 1. opens a connection to the server
// 2. sends a fixed number of messages per connection and prints the server's response
// 3. closes the connection
// 4. Repeats step 1, 2, 3
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log                   = logger.NewPretty("tcp-client")
	serverAddress         = flag.String("server-address", "", "address (ip:port) to connect to")
	requestPrefix         = "request:"
	connectRetryDelay     = 3 * time.Second
	nextMsgDelay          = 3 * time.Second
	msgCountPerConnection = 3
)

func main() {
	flag.Parse()

	connectionCounter := 0

	for {
		connectionCounter++
		conn, err := net.Dial("tcp", *serverAddress)
		if err != nil {
			fmt.Printf("Error connecting to %s: %s, retrying in %s time\n", *serverAddress, err, connectRetryDelay)
			time.Sleep(connectRetryDelay)
			continue
		}

		fmt.Printf("Started connection #%d to %s -----------------------\n", connectionCounter, *serverAddress)

		// Send as many messages determined by 'msgCountPerConnection' before creating a new connection
		response := bufio.NewReader(conn)
		for msgCounter := 1; msgCounter <= msgCountPerConnection; msgCounter++ {
			time.Sleep(nextMsgDelay)
			requestMsg := fmt.Sprintf("%s #connection=%d, #msg-counter=%d, msg=client hello\n", requestPrefix, connectionCounter, msgCounter)

			// write on the connection
			if bytesWritten, writeErr := conn.Write([]byte(requestMsg)); err != nil {
				log.Error().Err(writeErr).Msg("Write error")
				continue
			} else {
				fmt.Printf("Wrote %d bytes, msg written: [%s]\n", bytesWritten, requestMsg)
			}

			// read response from server
			responseMsg, err := response.ReadString(byte('\n'))
			switch err {
			case nil:
				fmt.Printf("Received response: [%s]\n", responseMsg)

			case io.EOF:
				fmt.Printf("Received EOF from server\n")

			default:
				// unexpected error
				fmt.Printf("Unexpected error: %s\n", err)
			}

			fmt.Println()
		}

		conn.Close() //nolint: errcheck,gosec
		fmt.Printf("Ended connection #%d ---------------------------\n\n", connectionCounter)
	}
}
