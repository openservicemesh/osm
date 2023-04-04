package cniserver

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/openservicemesh/osm/pkg/cni/config"
)

type qdisc struct {
	netns         string
	device        string
	managedClsact bool
}

type server struct {
	sync.Mutex
	unixSockPath string
	bpfMountPath string
	// qdiscs is for cleaning up all tc programs when exists
	// key: netns(inode), value: qdisc info
	qdiscs map[uint64]qdisc
	// listeners are the dummy sockets created for eBPF programs to fetch the current pod ip
	// key: netns(inode), value: net.Listener
	listeners map[uint64]net.Listener

	cniReady chan struct{}
	stop     chan struct{}
}

// NewServer returns a new CNI Server.
// the path this the unix path to listen.
func NewServer(unixSockPath string, bpfMountPath string, cniReady, stop chan struct{}) Server {
	if unixSockPath == "" {
		unixSockPath = path.Join(config.HostVarRun, "osm-cni.sock")
	}
	if bpfMountPath == "" {
		bpfMountPath = "/sys/fs/bpf"
	}
	return &server{
		unixSockPath: unixSockPath,
		bpfMountPath: bpfMountPath,
		qdiscs:       make(map[uint64]qdisc),
		listeners:    make(map[uint64]net.Listener),
		cniReady:     cniReady,
		stop:         stop,
	}
}

func (s *server) Start() error {
	if err := os.RemoveAll(s.unixSockPath); err != nil {
		log.Fatal().Err(err)
	}
	l, err := net.Listen("unix", s.unixSockPath)
	if err != nil {
		log.Fatal().Msgf("listen error:%v", err)
	}

	r := mux.NewRouter()
	r.Path(config.CNICreatePodURL).
		Methods("POST").
		HandlerFunc(s.PodCreated)

	r.Path(config.CNIDeletePodURL).
		Methods("POST").
		HandlerFunc(s.PodDeleted)

	ss := http.Server{
		Handler:           r,
		WriteTimeout:      10 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		go ss.Serve(l) // nolint: errcheck
		// TODO: unify all clean-up functions
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGABRT)
		select {
		case <-ch:
			s.Stop()
		case <-s.stop:
			s.Stop()
		}
		_ = ss.Shutdown(context.Background())
	}()

	s.installCNI()
	// wait for cni to be ready
	<-s.cniReady
	if err = s.checkAndRepairPodPrograms(); err != nil {
		log.Error().Msgf("Failed to check existing pods: %v", err)
	}
	return nil
}

func (s *server) installCNI() {
	install := newInstaller()
	go func() {
		if err := install.Run(context.TODO(), s.cniReady); err != nil {
			log.Error().Err(err)
			close(s.cniReady)
		}
		if err := install.Cleanup(); err != nil {
			log.Error().Msgf("Failed to clean up CNI: %v", err)
		}
	}()

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGABRT)
		<-ch
		if err := install.Cleanup(); err != nil {
			log.Error().Msgf("Failed to clean up CNI: %v", err)
		}
	}()
}

func (s *server) Stop() {
	log.Info().Msg("cni-server stop ...")
	s.cleanUpTC()
	close(s.stop)
}
