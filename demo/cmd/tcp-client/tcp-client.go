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
	logLevel              = flag.String("logLevel", "debug", "Log output level")
	serverAddress         = flag.String("server-address", "", "address (ip:port) to connect to")
	requestPrefix         = "request:"
	connectRetryDelay     = 3 * time.Second
	nextMsgDelay          = 3 * time.Second
	msgCountPerConnection = 3
)

func main() {
	flag.Parse()
	err := logger.SetLogLevel(*logLevel)
	if err != nil {
		log.Fatal().Msgf("Unknown log level: %s", *logLevel)
	}

	connectionCounter := 0

	for {
		connectionCounter++
		conn, err := net.Dial("tcp", *serverAddress)
		if err != nil {
			log.Error().Msgf("Error connecting to %s: %s, retrying in %s time", *serverAddress, err, connectRetryDelay)
			time.Sleep(connectRetryDelay)
			continue
		}

		log.Info().Msgf("Started connection #%d to %s -----------------------", connectionCounter, *serverAddress)

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
				log.Info().Msgf("Wrote %d bytes, msg written: [%s]\n", bytesWritten, requestMsg)
			}

			// read response from server
			responseMsg, err := response.ReadString(byte('\n'))
			switch err {
			case nil:
				log.Info().Msgf("Received response: [%s]", responseMsg)

			case io.EOF:
				log.Info().Msgf("Received EOF from server")

			default:
				// unexpected error
				log.Error().Err(err).Msgf("Unexpected error")
			}
		}

		//nolint: errcheck
		//#nosec G104
		conn.Close()
		log.Info().Msgf("Ended connection #%d ---------------------------", connectionCounter)
	}
}
