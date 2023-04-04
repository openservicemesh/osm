package cniserver

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/containernetworking/cni/pkg/skel"
)

func (s *server) PodCreated(w http.ResponseWriter, req *http.Request) {
	bs, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	args := skel.CmdArgs{}
	err = json.Unmarshal(bs, &args)
	log.Info().Msgf("cni called create with args: %+v", args)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	err = s.CmdAdd(&args)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *server) PodDeleted(w http.ResponseWriter, req *http.Request) {
	bs, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	args := skel.CmdArgs{}
	err = json.Unmarshal(bs, &args)
	log.Info().Msgf("cni called delete with args: %+v", args)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	err = s.CmdDelete(&args)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
