package main

import (
	//"errors"
	"fmt"
	"encoding/json"
	"net"
	"net/http"
	"github.com/openservicemesh/osm/pkg/logger"
	//"os"
	"google.golang.org/grpc"
	"sync"
)

const (
	httpPort = 8000
	grpcPort = 9000
)

var (
	log = logger.NewPretty("ext-authz")
)

func getRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got / request\n")
	log.Info().Msgf("Got / request")
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["message"] = "Status OK"
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error happened in JSON marshal. Err: %s", err)
	}
	w.Write(jsonResp)
	return
}

func startHTTPServer(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	http.HandleFunc("/", getRoot)
	log.Info().Msgf("Web server running on port %d", httpPort)
	err := http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil)
	log.Fatal().Err(err).Msgf("Failed to start HTTP server on port %d", httpPort)
}

func startGRPCServer(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))

	if err != nil {
		log.Error().Err(err).Msgf("Error starting gRPC server on %d", grpcPort)
	}

	s := grpc.NewServer()

	log.Info().Msgf("Setting up gRPC server on port %d", grpcPort)

	if err := s.Serve(lis); err != nil {
		log.Fatal().Err(err).Msgf("failed to serve: %v", err)
	}
}

func main() {
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go startHTTPServer(wg)
	go startGRPCServer(wg)
	wg.Wait()
}
