package helpers

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/cilium/ebpf"
)

// LoadProgs load ebpf progs
func LoadProgs(useCniMode, kernelTracing bool) error {
	if os.Getuid() != 0 {
		return fmt.Errorf("root user in required for this process or container")
	}
	cmd := exec.Command("make", "load")
	cmd.Env = os.Environ()
	if useCniMode {
		cmd.Env = append(cmd.Env, "CNI_MODE=true")
	}
	if !kernelTracing {
		cmd.Env = append(cmd.Env, "DEBUG=0")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if code := cmd.ProcessState.ExitCode(); code != 0 || err != nil {
		return fmt.Errorf("unexpected exit code: %d, err: %v", code, err)
	}
	return nil
}

// AttachProgs attach ebpf progs
func AttachProgs() error {
	if os.Getuid() != 0 {
		return fmt.Errorf("root user in required for this process or container")
	}
	cmd := exec.Command("make", "attach")
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if code := cmd.ProcessState.ExitCode(); code != 0 || err != nil {
		return fmt.Errorf("unexpected exit code: %d, err: %v", code, err)
	}
	return nil
}

// UnLoadProgs unload ebpf progs
func UnLoadProgs() error {
	cmd := exec.Command("make", "-k", "clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if code := cmd.ProcessState.ExitCode(); code != 0 || err != nil {
		return fmt.Errorf("unload unexpected exit code: %d, err: %v", code, err)
	}
	return nil
}

var (
	ingress *ebpf.Program
	egress  *ebpf.Program
)

// GetTrafficControlIngressProg returns tc ingress ebpf prog
func GetTrafficControlIngressProg() *ebpf.Program {
	if ingress == nil {
		err := initTrafficControlProgs()
		if err != nil {
			log.Error().Msgf("init tc prog filed: %v", err)
		}
	}
	return ingress
}

// GetTrafficControlEgressProg returns tc egress ebpf prog
func GetTrafficControlEgressProg() *ebpf.Program {
	if egress == nil {
		err := initTrafficControlProgs()
		if err != nil {
			log.Error().Msgf("init tc prog filed: %v", err)
		}
	}
	return egress
}

func initTrafficControlProgs() error {
	coll, err := ebpf.LoadCollectionSpec("bpf/osm_cni_tc_nat.o")
	if err != nil {
		return err
	}
	type progs struct {
		Ingress *ebpf.Program `ebpf:"osm_cni_tc_dnat"`
		Egress  *ebpf.Program `ebpf:"osm_cni_tc_snat"`
	}
	ps := progs{}
	err = coll.LoadAndAssign(&ps, &ebpf.CollectionOptions{
		MapReplacements: map[string]*ebpf.Map{
			"osm_pod_fib": GetPodFibMap(),
			"osm_nat_fib": GetNatFibMap(),
		},
	})
	if err != nil {
		return err
	}
	ingress = ps.Ingress
	egress = ps.Egress
	return nil
}
