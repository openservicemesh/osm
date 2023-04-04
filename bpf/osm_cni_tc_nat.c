#include "headers/helpers.h"
#include "headers/maps.h"
#include "headers/mesh.h"
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/in.h>
#include <linux/ip.h>
#include <linux/pkt_cls.h>
#include <linux/tcp.h>
#include <stddef.h>

__section("classifier_ingress") int osm_cni_tc_dnat(struct __sk_buff *skb)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    struct ethhdr *eth = (struct ethhdr *)data;
    if ((void *)(eth + 1) > data_end) {
        return TC_ACT_SHOT;
    }

    __u32 src_ip[4];
    __u32 dst_ip[4];
    struct tcphdr *tcph;
    __u32 csum_off;
    __u32 dport_off;

    switch (bpf_htons(eth->h_proto)) {
    case ETH_P_IP: {
        struct iphdr *iph = (struct iphdr *)(eth + 1);
        if ((void *)(iph + 1) > data_end) {
            return TC_ACT_SHOT;
        }
        if (iph->protocol == IPPROTO_IPIP) {
            iph = ((void *)iph + iph->ihl * 4);
            if ((void *)(iph + 1) > data_end) {
                return TC_ACT_OK;
            }
        }
        if (iph->protocol != IPPROTO_TCP) {
            return TC_ACT_OK;
        }
        set_ipv4(src_ip, iph->saddr);
        set_ipv4(dst_ip, iph->daddr);
        tcph = (struct tcphdr *)(iph + 1);
        csum_off =
            ETH_HLEN + sizeof(struct iphdr) + offsetof(struct tcphdr, check);
        dport_off =
            ETH_HLEN + sizeof(struct iphdr) + offsetof(struct tcphdr, dest);
        break;
    }
    default:
        return TC_ACT_OK;
    }

    if ((void *)(tcph + 1) > data_end) {
        return TC_ACT_SHOT;
    }
    __u16 in_port = bpf_htons(IN_REDIRECT_PORT);
    if (tcph->syn && !tcph->ack) {
        // first packet
        if (tcph->dest == in_port) {
            // same node, already rewrite dest port by connect. bypass.
            debugf("osm_cni_tc_nat [ingress]: already dnat");
            return TC_ACT_OK;
        }
        // ingress without osm_cni_grp_connect
        struct pod_config *pod = bpf_map_lookup_elem(&osm_pod_fib, dst_ip);
        if (!pod) {
            // dest ip is not on this node or not injected sidecar.
            debugf("osm_cni_tc_nat [ingress]: pod not found, bypassed");
            return TC_ACT_OK;
        }
        if (bpf_htons(tcph->dest) == pod->status_port) {
            return TC_ACT_OK;
        }
        int exclude = 0;
        IS_EXCLUDE_PORT(pod->exclude_in_ports, tcph->dest, &exclude);
        if (exclude) {
            debugf("osm_cni_tc_nat [ingress]: ignored dest port by "
                   "exclude_in_ports, ip: %pI4/%pI6c, port: %d",
                   &dst_ip[3], dst_ip, bpf_htons(tcph->dest));
            return TC_ACT_OK;
        }
        int include = 0;
        IS_INCLUDE_PORT(pod->include_in_ports, tcph->dest, &include);
        if (!include) {
            debugf("osm_cni_tc_nat [ingress]: ignored dest port by "
                   "include_in_ports, ip: %pI4/%pI6c, port: %d",
                   &dst_ip[3], dst_ip, bpf_htons(tcph->dest));
            return TC_ACT_OK;
        }

        struct pair p;
        memset(&p, 0, sizeof(p));
        set_ipv6(p.sip, src_ip);
        set_ipv6(p.dip, dst_ip);
        p.sport = tcph->source;
        p.dport = in_port;

        __u16 dst_port = tcph->dest;
        struct origin_info origin;
        memset(&origin, 0, sizeof(origin));
        set_ipv6(origin.ip, dst_ip);
        origin.port = dst_port;
        origin.flags = TC_ORIGIN_FLAG;
        bpf_map_update_elem(&osm_nat_fib, &p, &origin, BPF_NOEXIST);

        bpf_l4_csum_replace(skb, csum_off, dst_port, in_port, sizeof(dst_port));
        bpf_skb_store_bytes(skb, dport_off, &in_port, sizeof(in_port), 0);
        debugf("osm_cni_tc_nat [ingress]: first dnat");
    } else {
        // request
        struct pair p;
        memset(&p, 0, sizeof(p));
        set_ipv6(p.sip, src_ip);
        set_ipv6(p.dip, dst_ip);
        p.sport = tcph->source;
        p.dport = in_port;
        struct origin_info *origin = bpf_map_lookup_elem(&osm_nat_fib, &p);
        if (!origin) {
            return TC_ACT_OK;
        }
        if (!(origin->flags & TC_ORIGIN_FLAG)) {
            // not tc origin
            debugf("osm_cni_tc_nat [ingress]: no tc origin flag");
            return TC_ACT_OK;
        }
        __u16 dst_port = tcph->dest;
        bpf_l4_csum_replace(skb, csum_off, dst_port, in_port, sizeof(dst_port));
        bpf_skb_store_bytes(skb, dport_off, &in_port, sizeof(in_port), 0);
        debugf("osm_cni_tc_nat [ingress]: dnat");
    }
    return TC_ACT_OK;
}

__section("classifier_egress") int osm_cni_tc_snat(struct __sk_buff *skb)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    struct ethhdr *eth = (struct ethhdr *)data;
    if ((void *)(eth + 1) > data_end) {
        return TC_ACT_SHOT;
    }

    __u32 src_ip[4];
    __u32 dst_ip[4];
    struct tcphdr *tcph;
    __u32 csum_off;
    __u32 sport_off;

    switch (bpf_htons(eth->h_proto)) {
    case ETH_P_IP: {
        struct iphdr *iph = (struct iphdr *)(eth + 1);
        if ((void *)(iph + 1) > data_end) {
            return TC_ACT_SHOT;
        }
        if (iph->protocol == IPPROTO_IPIP) {
            iph = ((void *)iph + iph->ihl * 4);
            if ((void *)(iph + 1) > data_end) {
                return TC_ACT_OK;
            }
        }
        if (iph->protocol != IPPROTO_TCP) {
            return TC_ACT_OK;
        }
        set_ipv4(src_ip, iph->saddr);
        set_ipv4(dst_ip, iph->daddr);
        tcph = (struct tcphdr *)(iph + 1);
        csum_off =
            ETH_HLEN + sizeof(struct iphdr) + offsetof(struct tcphdr, check);
        sport_off =
            ETH_HLEN + sizeof(struct iphdr) + offsetof(struct tcphdr, source);
        break;
    }
    default:
        return TC_ACT_OK;
    }

    if ((void *)(tcph + 1) > data_end) {
        return TC_ACT_SHOT;
    }
    __u16 in_port = bpf_htons(IN_REDIRECT_PORT);
    if (tcph->source != in_port) {
        // debugf("osm_cni_tc_nat[egress]: no need to rewrite src port,
        // bypassed");
        return TC_ACT_OK;
    }
    struct pair p;
    memset(&p, 0, sizeof(p));
    set_ipv6(p.dip, src_ip);
    set_ipv6(p.sip, dst_ip);
    p.dport = tcph->source;
    p.sport = tcph->dest;
    struct origin_info *origin = bpf_map_lookup_elem(&osm_nat_fib, &p);
    if (!origin) {
        // not exists
        debugf("osm_cni_tc_nat [egress]: resp origin not found");
        return TC_ACT_OK;
    }
    if (!(origin->flags & TC_ORIGIN_FLAG)) {
        // not tc origin
        printk("osm_cni_tc_nat [egress]: resp origin flags %x error",
               origin->flags);
        return TC_ACT_OK;
    }
    if (tcph->fin && tcph->ack) {
        // todo delete key
        debugf("osm_cni_tc_nat [egress]: original deleted");
        bpf_map_delete_elem(&osm_nat_fib, &p);
    }
    __u16 src_port = origin->port;
    bpf_l4_csum_replace(skb, csum_off, in_port, src_port, sizeof(src_port));
    bpf_skb_store_bytes(skb, sport_off, &src_port, sizeof(src_port), 0);
    debugf("osm_cni_tc_nat [egress]: snat");
    return TC_ACT_OK;
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
