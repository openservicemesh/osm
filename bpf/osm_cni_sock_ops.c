#include "headers/helpers.h"
#include "headers/maps.h"
#include "headers/mesh.h"
#include <linux/bpf.h>

static inline int osm_cni_sockops_ipv4(struct bpf_sock_ops *skops)
{
    struct pair p;
    memset(&p, 0, sizeof(p));
    set_ipv4(p.sip, skops->local_ip4);
    p.sport = bpf_htons(skops->local_port);
    set_ipv4(p.dip, skops->remote_ip4);
    p.dport = skops->remote_port >> 16;

    __u64 cookie = bpf_get_socket_cookie_ops(skops);
    struct origin_info *dst = bpf_map_lookup_elem(&osm_cki_fib, &cookie);
    if (dst) {
        struct origin_info dd = *dst;
        if (!(dd.flags & 1)) {
            __u32 pid = dd.pid;
            // process ip not detected
            if (skops->local_ip4 == sidecar_ip ||
                skops->local_ip4 == skops->remote_ip4) {
                // sidecar to local
                __u32 ip = skops->remote_ip4;
                debugf("osm_cni_sockops_ipv4 [Sidecar->Local] detected process "
                       "%d's ip is %pI4",
                       pid, &ip);
                bpf_map_update_elem(&osm_proc_fib, &pid, &ip, BPF_ANY);
                if (skops->remote_port >> 16 == bpf_htons(IN_REDIRECT_PORT)) {
                    printk("incorrect connection: cookie=%d", cookie);
                    return 1;
                }
            } else {
                // sidecar to sidecar
                __u32 ip = skops->local_ip4;
                bpf_map_update_elem(&osm_proc_fib, &pid, &ip, BPF_ANY);
                debugf("osm_cni_sockops_ipv4 [Sidecar->Sidecar] detected "
                       "process %d's ip is %pI4",
                       pid, &ip);
            }
        }
#ifdef DEBUG
        __u32 remote_ip4 = get_ipv4(p.dip);
        __u32 local_ip4 = get_ipv4(p.sip);
        debugf("osm_cni_sockops_ipv4 [established] remote_ip4: %pI4 -> "
               "local_ip4: %pI4",
               &remote_ip4, &local_ip4);
        debugf("osm_cni_sockops_ipv4 [established] remote_port: %d -> "
               "local_port: %d",
               bpf_htons(p.dport), skops->local_port);
#endif
        // get_sockopts can read pid and cookie,
        // we should write a new map named osm_nat_fib
        bpf_map_update_elem(&osm_nat_fib, &p, &dd, BPF_ANY);
        bpf_sock_hash_update(skops, &osm_sock_fib, &p, BPF_NOEXIST);
    } else if (skops->local_port == OUT_REDIRECT_PORT ||
               skops->local_port == IN_REDIRECT_PORT ||
               skops->remote_ip4 == sidecar_ip) {
#ifdef DEBUG
        __u32 remote_ip4 = get_ipv4(p.dip);
        __u32 local_ip4 = get_ipv4(p.sip);
        debugf("osm_cni_sockops_ipv4 [established] remote_ip4: %pI4 -> "
               "local_ip4: %pI4",
               &remote_ip4, &local_ip4);
        debugf("osm_cni_sockops_ipv4 [established] remote_port: %d -> "
               "local_port: %d",
               bpf_htons(p.dport), skops->local_port);
#endif
        bpf_sock_hash_update(skops, &osm_sock_fib, &p, BPF_NOEXIST);
    }
    return 0;
}

__section("sockops") int osm_cni_sock_ops(struct bpf_sock_ops *skops)
{
    switch (skops->op) {
    case BPF_SOCK_OPS_PASSIVE_ESTABLISHED_CB:
    case BPF_SOCK_OPS_ACTIVE_ESTABLISHED_CB:
        switch (skops->family) {
        case 2:
            // AF_INET, we don't include socket.h, because it may
            // cause an import error.
            return osm_cni_sockops_ipv4(skops);
        }
        return 0;
    }
    return 0;
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
