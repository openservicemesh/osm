package cniserver

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/cilium/ebpf"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"golang.org/x/sys/unix"

	"github.com/openservicemesh/osm/pkg/cni/config"
	"github.com/openservicemesh/osm/pkg/cni/controller/helpers"
	"github.com/openservicemesh/osm/pkg/cni/ns"
	"github.com/openservicemesh/osm/pkg/cni/plugin"
	"github.com/openservicemesh/osm/pkg/cni/util"
)

func getMarkKeyOfNetns(netns string) uint32 {
	// todo check conflict?
	algorithm := fnv.New32a()
	_, _ = algorithm.Write([]byte(netns))
	return algorithm.Sum32()
}

func (s *server) CmdAdd(args *skel.CmdArgs) (err error) {
	defer func() {
		if e := recover(); e != nil {
			msg := fmt.Sprintf("osm-cni panic during cmdAdd: %v\n%v", e, string(debug.Stack()))
			if err != nil {
				// If we're recovering and there was also an error, then we need to
				// present both.
				msg = fmt.Sprintf("%s: %v", msg, err)
			}
			err = fmt.Errorf(msg)
		}
		if err != nil {
			log.Error().Msgf("osm-cni cmdAdd error: %v", err)
		}
	}()
	k8sArgs := plugin.K8sArgs{}
	if err := types.LoadArgs(args.Args, &k8sArgs); err != nil {
		return err
	}
	netns, err := ns.GetNS("/host" + args.Netns)
	if err != nil {
		log.Error().Msgf("get ns %s error", args.Netns)
		return err
	}

	err = netns.Do(func(_ ns.NetNS) error {
		// listen on 15050
		if err := s.buildListener(netns.Path()); err != nil {
			return err
		}
		// attach tc to the device
		if len(args.IfName) != 0 {
			return s.attachTC(netns.Path(), args.IfName)
		}
		// interface not specified, should not happen?
		ifaces, _ := net.Interfaces()
		for _, iface := range ifaces {
			if (iface.Flags&net.FlagLoopback) == 0 && (iface.Flags&net.FlagUp) != 0 {
				return s.attachTC(netns.Path(), iface.Name)
			}
		}
		return fmt.Errorf("device not found for %s", args.Netns)
	})
	if err != nil {
		log.Error().Msgf("CmdAdd failed for %s: %v", args.Netns, err)
		return err
	}
	return err
}

func (s *server) CmdDelete(args *skel.CmdArgs) (err error) {
	k8sArgs := plugin.K8sArgs{}
	if err := types.LoadArgs(args.Args, &k8sArgs); err != nil {
		return err
	}
	netns := "/host" + args.Netns
	inode, err := util.Inode(netns)
	if err != nil {
		return err
	}
	s.Lock()

	delete(s.qdiscs, inode)
	delete(s.listeners, inode)

	s.Unlock()
	m, err := ebpf.LoadPinnedMap(path.Join(s.bpfMountPath, "osm_mark_fib"), &ebpf.LoadPinOptions{})
	if err != nil {
		return err
	}
	key := getMarkKeyOfNetns(args.Netns)
	return m.Delete(key)
}

// listen on 15050
func (s *server) buildListener(netns string) error {
	inode, err := util.Inode(netns)
	if err != nil {
		return err
	}
	var addrs []net.Addr
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if (iface.Flags&net.FlagLoopback) != 0 || (iface.Flags&net.FlagUp) == 0 {
			continue
		}
		ifAddrs, err := iface.Addrs()
		if err != nil || len(ifAddrs) == 0 {
			continue
		}
		addrs = append(addrs, ifAddrs...)
	}
	if len(addrs) == 0 {
		log.Error().Msgf("no ip address for %s", netns)
		return nil
	}
	if len(addrs) != 1 {
		log.Warn().Msgf("get ip address for %s: res: %v, only support single ip address", netns, addrs)
	}

	lc := s.listenConfig(addrs[0], netns)
	var l net.Listener
	l, err = lc.Listen(context.Background(), "tcp", "0.0.0.0:15050")
	if err != nil {
		return err
	}

	s.Lock()
	// keep the listener, otherwise it will be GCed
	s.listeners[inode] = l
	s.Unlock()
	return nil
}

func (s *server) listenConfig(addr net.Addr, netns string) net.ListenConfig {
	return net.ListenConfig{
		Control: func(network, address string, conn syscall.RawConn) error {
			var operr error
			if err := conn.Control(func(fd uintptr) {
				m, err := ebpf.LoadPinnedMap(path.Join(s.bpfMountPath, "osm_mark_fib"), &ebpf.LoadPinOptions{})
				if err != nil {
					operr = err
					return
				}
				var ip unsafe.Pointer
				switch v := addr.(type) { // todo instead of hash
				case *net.IPNet: // nolint: typecheck
					ip, err = util.IP2Pointer(v.IP.String())
				case *net.IPAddr: // nolint: typecheck
					ip, err = util.IP2Pointer(v.String())
				}
				if err != nil {
					operr = err
					return
				}
				key := getMarkKeyOfNetns(netns)
				operr = m.Update(key, ip, ebpf.UpdateAny)
				if operr != nil {
					return
				}
				operr = syscall.SetsockoptInt(int(fd), unix.SOL_SOCKET, ns.SoMark, int(key))
			}); err != nil {
				return err
			}
			return operr
		},
	}
}

func (s *server) checkAndRepairPodPrograms() error {
	hostProc, err := os.ReadDir(config.HostProc)
	if err != nil {
		return err
	}
	for _, f := range hostProc {
		if _, err = strconv.Atoi(f.Name()); err == nil {
			pid := f.Name()
			if skipListening(pid) {
				// ignore non-injected pods
				log.Debug().Msgf("skip listening for pid(%s)", pid)
				continue
			}
			np := fmt.Sprintf("%s/%s/ns/net", config.HostProc, pid)
			netns, err := ns.GetNS(np)
			if err != nil {
				log.Error().Msgf("Failed to get ns for %s, error: %v", np, err)
				continue
			}
			if err = netns.Do(func(_ ns.NetNS) error {
				log.Info().Msgf("build listener for pid(%s)", pid)
				// listen on 15050
				if err := s.buildListener(netns.Path()); err != nil {
					return err
				}
				// attach tc to the device
				ifaces, _ := net.Interfaces()
				for _, iface := range ifaces {
					if (iface.Flags&net.FlagLoopback) == 0 && (iface.Flags&net.FlagUp) != 0 {
						err := s.attachTC(netns.Path(), iface.Name)
						if err != nil {
							log.Error().Msgf("attach tc for %s of %s error: %v", iface.Name, netns.Path(), err)
						}
						return err
					}
				}
				return fmt.Errorf("device not found for pid(%s)", pid)
			}); err != nil {
				if errors.Is(err, syscall.EADDRINUSE) {
					// skip if it has listened on 15050
					continue
				}
				return err
			}
		}
	}
	return nil
}

func skipListening(pid string) bool {
	findStr := func(path string, str []byte) bool {
		//#nosec G304
		f, _ := os.Open(path)
		defer func(f *os.File) {
			_ = f.Close()
		}(f)
		sc := bufio.NewScanner(f)
		sc.Split(bufio.ScanLines)
		for sc.Scan() {
			if bytes.Contains(sc.Bytes(), str) {
				return true
			}
		}
		return false
	}

	conn4 := fmt.Sprintf("%s/%s/net/tcp", config.HostProc, pid)
	return !findStr(conn4, []byte(fmt.Sprintf(": %0.8d:%0.4X %0.8d:%0.4X 0A", 0, 15001, 0, 0)))
}

func uint32Ptr(v uint32) *uint32 {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func (s *server) attachTC(netns, dev string) error {
	// already in netns
	inode, err := util.Inode(netns)
	if err != nil {
		return err
	}
	iface, err := net.InterfaceByName(dev)
	if err != nil {
		log.Error().Msgf("get iface error: %v", err)
		return err
	}
	rtnl, err := tc.Open(&tc.Config{})
	if err != nil {
		log.Error().Msgf("open rtnl error: %v", err)
		return err
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
			log.Error().Msgf("could not close rtnetlink socket: %v\n", err)
		}
	}()
	qdiscs, err := rtnl.Qdisc().Get()
	if err != nil {
		log.Error().Msgf("get qdisc error: %v", err)
		return err
	}
	find := false
	for _, qdisc := range qdiscs {
		if qdisc.Kind == "clsact" && qdisc.Ifindex == uint32(iface.Index) {
			find = true
			break
		}
	}
	if !find {
		// init clasact if not exists
		err := rtnl.Qdisc().Add(&tc.Object{
			Msg: tc.Msg{
				Family:  unix.AF_UNSPEC,
				Ifindex: uint32(iface.Index),
				Handle:  core.BuildHandle(0xFFFF, 0x0000),
				Parent:  tc.HandleIngress,
			},
			Attribute: tc.Attribute{
				Kind: "clsact",
			},
		})
		if err != nil {
			return err
		}
	}
	ing := helpers.GetTrafficControlIngressProg()
	if ing == nil {
		return fmt.Errorf("can not get ingress prog")
	}

	filter := tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(iface.Index),
			// Handle:  0,
			Parent: 0xFFFFFFF2, // ingress
			Info: core.BuildHandle(
				66,     // prio
				0x0300, // protocol
			),
		},
		Attribute: tc.Attribute{
			Kind: "bpf",
			BPF: &tc.Bpf{
				FD:    uint32Ptr(uint32(ing.FD())),
				Name:  stringPtr("osm_cni_tc_dnat"),
				Flags: uint32Ptr(0x1),
			},
		},
	}
	if err := rtnl.Filter().Add(&filter); err != nil {
		return err
	}
	egress := helpers.GetTrafficControlEgressProg()
	if ing == nil {
		return fmt.Errorf("can not get egress prog")
	}

	filter = tc.Object{
		Msg: tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(iface.Index),
			// Handle:  0,
			Parent: 0xFFFFFFF3, // egress
			Info: core.BuildHandle(
				66,     // prio
				0x0300, // protocol
			),
		},
		Attribute: tc.Attribute{
			Kind: "bpf",
			BPF: &tc.Bpf{
				FD:    uint32Ptr(uint32(egress.FD())),
				Name:  stringPtr("osm_cni_tc_snat"),
				Flags: uint32Ptr(0x1),
			},
		},
	}
	if err := rtnl.Filter().Add(&filter); err != nil {
		return err
	}

	s.Lock()
	s.qdiscs[inode] = qdisc{
		netns:         netns,
		device:        dev,
		managedClsact: !find,
	}
	s.Unlock()
	return nil
}

func (s *server) cleanUpTC() {
	s.Lock()
	defer s.Unlock()
	for _, q := range s.qdiscs {
		netns, err := ns.GetNS(q.netns)
		if err != nil {
			log.Error().Msgf("Failed to get ns for %s, error: %v", q.netns, err)
			continue
		}
		if err = netns.Do(func(_ ns.NetNS) error {
			iface, err := net.InterfaceByName(q.device)
			if err != nil {
				return err
			}
			rtnl, err := tc.Open(&tc.Config{})
			if err != nil {
				return err
			}
			defer func() {
				if err := rtnl.Close(); err != nil {
					log.Error().Msgf("could not close rtnetlink socket: %v\n", err)
				}
			}()
			if q.managedClsact {
				err := rtnl.Qdisc().Delete(&tc.Object{
					Msg: tc.Msg{
						Family:  unix.AF_UNSPEC,
						Ifindex: uint32(iface.Index),
						Handle:  core.BuildHandle(0xFFFF, 0x0000),
						Parent:  tc.HandleIngress,
					},
					Attribute: tc.Attribute{
						Kind: "clsact",
					},
				})
				if err != nil {
					log.Error().Msgf("error remove clsact: ns: %s, dev: %s, err: %v", q.netns, q.device, err)
					// if remove clsact error, rollback to remove filter
				} else {
					return nil
				}
			}
			filter := tc.Object{
				Msg: tc.Msg{
					Family:  unix.AF_UNSPEC,
					Ifindex: uint32(iface.Index),
					Parent:  0xFFFFFFF2,
					Info: core.BuildHandle(
						66,     // prio
						0x0300, // protocol
					),
				},
			}
			_ = rtnl.Filter().Delete(&filter)
			filter = tc.Object{
				Msg: tc.Msg{
					Family:  unix.AF_UNSPEC,
					Ifindex: uint32(iface.Index),
					Parent:  0xFFFFFFF3,
					Info: core.BuildHandle(
						66,     // prio
						0x0300, // protocol
					),
				},
			}
			return rtnl.Filter().Delete(&filter)
		}); err != nil {
			log.Error().Msgf("Failed to clean up tc for %s, error: %v", q.netns, err)
		}
	}
}
