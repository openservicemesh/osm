#pragma once

#include "helpers.h"
#include "maps.h"
#include "mesh.h"
#include <linux/bpf.h>

#ifndef ENABLE_CNI_MODE
#define ENABLE_CNI_MODE 0
#endif

#define DNS_CAPTURE_PORT_FLAG (1 << 1)

// get_current_cgroup_info return 1 if succeed, 0 for error
static inline int get_current_cgroup_info(void *ctx,
                                          struct cgroup_info *cg_info)
{
    if (!cg_info) {
        printk("cg_info can not be NULL");
        return 0;
    }
    __u64 cgroup_id = bpf_get_current_cgroup_id();
    void *info = bpf_map_lookup_elem(&osm_cgr_fib, &cgroup_id);
    if (!info) {
        struct cgroup_info _default = {
            .id = cgroup_id,
            .is_in_mesh = 0,
            .cgroup_ip = {0, 0, 0, 0},
            .flags = 0,
            .detected_flags = 0,
        };
#if ENABLE_CNI_MODE
        // get ip addresses of current pod/ns.
        struct bpf_sock_tuple tuple = {};
        tuple.ipv4.dport = bpf_htons(SOCK_IP_MARK_PORT);
        tuple.ipv4.daddr = 0;
        struct bpf_sock *s = bpf_sk_lookup_tcp(ctx, &tuple, sizeof(tuple.ipv4),
                                               BPF_F_CURRENT_NETNS, 0);
        if (s) {
            __u32 curr_ip_mark = s->mark;
            bpf_sk_release(s);
            // get ip addresses of current pod/ns.
            __u32 *ip =
                (__u32 *)bpf_map_lookup_elem(&osm_mark_fib, &curr_ip_mark);
            if (!ip) {
                debugf("get ip for mark 0x%x error", curr_ip_mark);
            } else {
                set_ipv6(_default.cgroup_ip, ip); // network order
            }
            // in mesh
            _default.is_in_mesh = 1;
        } else {
            // not in mesh
            _default.is_in_mesh = 0;
        }
#else
        // not checked ever
        if (!is_port_listen_current_ns(ctx, ip_zero, OUT_REDIRECT_PORT)) {
            // not in mesh
            _default.is_in_mesh = 0;
            debugf("can not get port listen for cgroup(%ld)", cgroup_id);
        } else {
            _default.is_in_mesh = 1;
        }
#endif
        if (bpf_map_update_elem(&osm_cgr_fib, &cgroup_id, &_default, BPF_ANY)) {
            printk("update osm_cgr_fib of cgroup(%ld) error", cgroup_id);
            return 0;
        }
        *cg_info = _default;
    } else {
        *cg_info = *(struct cgroup_info *)info;
    }
    return 1;
}