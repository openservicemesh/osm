#pragma once

#include <asm-generic/int-ll64.h>
#include <linux/bpf.h>
#include <linux/bpf_common.h>
#include <linux/in.h>
#include <linux/in6.h>
#include <linux/socket.h>
#include <linux/swab.h>
#include <linux/types.h>

#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__
#define bpf_ntohs(x) __builtin_bswap16(x)
#define bpf_ntohl(x) __builtin_bswap32(x)
#define bpf_htons(x) __builtin_bswap16(x)
#define bpf_htonl(x) __builtin_bswap32(x)
#elif __BYTE_ORDER__ == __ORDER_BIG_ENDIAN__
#define bpf_htons(x) (x)
#define bpf_htonl(x) (x)
#define bpf_ntohs(x) (x)
#define bpf_ntohl(x) (x)
#else
#error "__BYTE_ORDER__ error"
#endif

#ifndef __section
#define __section(NAME) __attribute__((section(NAME), used))
#endif

#define PIN_GLOBAL_NS 2

struct bpf_elf_map {
    __u32 type;
    __u32 size_key;
    __u32 size_value;
    __u32 max_elem;
    __u32 flags;
    __u32 id;
    __u32 pinning;
};

static __u64 (*bpf_get_current_pid_tgid)() = (void *)
    BPF_FUNC_get_current_pid_tgid;

static __u64 (*bpf_get_current_uid_gid)() = (void *)
    BPF_FUNC_get_current_uid_gid;

static __u64 (*bpf_get_current_cgroup_id)() = (void *)
    BPF_FUNC_get_current_cgroup_id;

static void (*bpf_trace_printk)(const char *fmt, int fmt_size,
                                ...) = (void *)BPF_FUNC_trace_printk;

static __u64 (*bpf_get_current_comm)(void *buf, __u32 size_of_buf) = (void *)
    BPF_FUNC_get_current_comm;

static __u64 (*bpf_get_socket_cookie_ops)(struct bpf_sock_ops *skops) = (void *)
    BPF_FUNC_get_socket_cookie;

static __u64 (*bpf_get_socket_cookie_addr)(struct bpf_sock_addr *ctx) = (void *)
    BPF_FUNC_get_socket_cookie;

static void *(*bpf_map_lookup_elem)(struct bpf_elf_map *map, const void *key) =
    (void *)BPF_FUNC_map_lookup_elem;

static __u64 (*bpf_map_update_elem)(struct bpf_elf_map *map, const void *key,
                                    const void *value, __u64 flags) = (void *)
    BPF_FUNC_map_update_elem;

static __u64 (*bpf_map_delete_elem)(struct bpf_elf_map *map, const void *key) =
    (void *)BPF_FUNC_map_delete_elem;

static struct bpf_sock *(*bpf_sk_lookup_tcp)(
    void *ctx, struct bpf_sock_tuple *tuple, __u32 tuple_size, __u64 netns,
    __u64 flags) = (void *)BPF_FUNC_sk_lookup_tcp;

static struct bpf_sock *(*bpf_sk_lookup_udp)(
    void *ctx, struct bpf_sock_tuple *tuple, __u32 tuple_size, __u64 netns,
    __u64 flags) = (void *)BPF_FUNC_sk_lookup_udp;

static long (*bpf_sk_release)(struct bpf_sock *sock) = (void *)
    BPF_FUNC_sk_release;

static long (*bpf_sock_hash_update)(
    struct bpf_sock_ops *skops, struct bpf_elf_map *map, void *key,
    __u64 flags) = (void *)BPF_FUNC_sock_hash_update;

static long (*bpf_msg_redirect_hash)(
    struct sk_msg_md *md, struct bpf_elf_map *map, void *key,
    __u64 flags) = (void *)BPF_FUNC_msg_redirect_hash;

static long (*bpf_bind)(struct bpf_sock_addr *ctx, struct sockaddr_in *addr,
                        int addr_len) = (void *)BPF_FUNC_bind;

static long (*bpf_l4_csum_replace)(struct __sk_buff *skb, __u32 offset,
                                   __u64 from, __u64 to, __u64 flags) = (void *)
    BPF_FUNC_l4_csum_replace;

static int (*bpf_skb_load_bytes)(void *ctx, int off, void *to,
                                 int len) = (void *)BPF_FUNC_skb_load_bytes;

static long (*bpf_skb_store_bytes)(struct __sk_buff *skb, __u32 offset,
                                   const void *from, __u32 len, __u64 flags) =
    (void *)BPF_FUNC_skb_store_bytes;

#ifdef PRINTNL
#define PRINT_SUFFIX "\n"
#else
#define PRINT_SUFFIX ""
#endif

#ifndef printk
#define printk(fmt, ...)                                                       \
    ({                                                                         \
        char ____fmt[] = fmt PRINT_SUFFIX;                                     \
        bpf_trace_printk(____fmt, sizeof(____fmt), ##__VA_ARGS__);             \
    })
#endif

#ifndef DEBUG
// do nothing
#define debugf(fmt, ...) ({})
#else
// only print traceing in debug mode
#ifndef debugf
#define debugf(fmt, ...)                                                       \
    ({                                                                         \
        char ____fmt[] = "[debug] " fmt PRINT_SUFFIX;                          \
        bpf_trace_printk(____fmt, sizeof(____fmt), ##__VA_ARGS__);             \
    })
#endif

#endif

#ifndef memset
#define memset(dst, src, len) __builtin_memset(dst, src, len)
#endif

static const __u32 ip_zero = 0;
// 127.0.0.1 (network order)
static const __u32 localhost = 127 + (1 << 24);

static inline __u32 get_ipv4(__u32 *ip) { return ip[3]; }

static inline void set_ipv4(__u32 *dst, __u32 src)
{
    memset(dst, 0, sizeof(__u32) * 3);
    dst[3] = src;
}

static inline int is_port_listen_current_ns(void *ctx, __u32 ip, __u16 port)
{

    struct bpf_sock_tuple tuple = {};
    tuple.ipv4.dport = bpf_htons(port);
    tuple.ipv4.daddr = ip;
    struct bpf_sock *s = bpf_sk_lookup_tcp(ctx, &tuple, sizeof(tuple.ipv4),
                                           BPF_F_CURRENT_NETNS, 0);
    if (s) {
        bpf_sk_release(s);
        return 1;
    }
    return 0;
}

static inline int is_port_listen_udp_current_ns(void *ctx, __u32 ip, __u16 port)
{
    struct bpf_sock_tuple tuple = {};
    tuple.ipv4.dport = bpf_htons(port);
    tuple.ipv4.daddr = ip;
    struct bpf_sock *s = bpf_sk_lookup_udp(ctx, &tuple, sizeof(tuple.ipv4),
                                           BPF_F_CURRENT_NETNS, 0);
    if (s) {
        bpf_sk_release(s);
        return 1;
    }
    return 0;
}

static const __u32 ip_zero6[4] = {0, 0, 0, 0};
// ::1 (network order)
static const __u32 localhost6[4] = {0, 0, 0, 1 << 24};

static inline void set_ipv6(__u32 *dst, __u32 *src)
{
    dst[0] = src[0];
    dst[1] = src[1];
    dst[2] = src[2];
    dst[3] = src[3];
}

static inline int ipv6_equal(__u32 *a, __u32 *b)
{
    return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3];
}

static inline int is_port_listen_current_ns6(void *ctx, __u32 *ip, __u16 port)
{
    struct bpf_sock_tuple tuple = {};
    tuple.ipv6.dport = bpf_htons(port);
    set_ipv6(tuple.ipv6.daddr, ip);
    struct bpf_sock *s = bpf_sk_lookup_tcp(ctx, &tuple, sizeof(tuple.ipv6),
                                           BPF_F_CURRENT_NETNS, 0);
    if (s) {
        bpf_sk_release(s);
        return 1;
    }
    return 0;
}

static inline int is_port_listen_udp_current_ns6(void *ctx, __u32 *ip,
                                                 __u16 port)
{
    struct bpf_sock_tuple tuple = {};
    tuple.ipv6.dport = bpf_htons(port);
    set_ipv6(tuple.ipv6.daddr, ip);
    struct bpf_sock *s = bpf_sk_lookup_udp(ctx, &tuple, sizeof(tuple.ipv6),
                                           BPF_F_CURRENT_NETNS, 0);
    if (s) {
        bpf_sk_release(s);
        return 1;
    }
    return 0;
}

struct origin_info {
    __u32 ip[4];
    __u32 pid;
    __u16 port;
    // last bit means that ip of process is detected.
    __u16 flags;
};

struct pair {
    __u32 sip[4];
    __u32 dip[4];
    __u16 sport;
    __u16 dport;
};

struct cgroup_info {
    __u64 id;
    __u32 is_in_mesh;
    __u32 cgroup_ip[4];
    // We can't specify which ports are listened to here, so we open up a flags,
    // user-defined. E.g, for those who wish to determine if port 15001 is
    // listened to, we can customize a flag, `IS_LISTEN_15001 = 1 << 2`, which
    // we can subsequently detect by `flags & IS_LISTEN_15001`.
    __u16 flags;
    // detected_flags is used to determine if this operation has ever been
    // performed. if `flags & IS_LISTEN_15001` is false but `detected_flags &
    // IS_LISTEN_15001` is true, that means real true, we do not need recheck.
    // but if `detected_flags & IS_LISTEN_15001` is false, that probably means
    // we haven't tested it and need to retest it.
    __u16 detected_flags;
};

#define MAX_ITEM_LEN 20

struct cidr {
    __u32 net; // network order
    __u8 mask;
    __u8 __pad[3];
};

static inline int is_in_cidr(struct cidr *c, __u32 ip)
{
    return (bpf_htonl(c->net) >> (32 - c->mask)) ==
           bpf_htonl(ip) >> (32 - c->mask);
}

struct pod_config {
    __u16 status_port;
    __u16 __pad;
    struct cidr exclude_out_ranges[MAX_ITEM_LEN];
    struct cidr include_out_ranges[MAX_ITEM_LEN];
    __u16 include_in_ports[MAX_ITEM_LEN];
    __u16 include_out_ports[MAX_ITEM_LEN];
    __u16 exclude_in_ports[MAX_ITEM_LEN];
    __u16 exclude_out_ports[MAX_ITEM_LEN];
};

#define IS_EXCLUDE_PORT(ITEM, PORT, RET)                                       \
    do {                                                                       \
        *RET = 0;                                                              \
        for (int i = 0; i < MAX_ITEM_LEN && ITEM[i] != 0; i++) {               \
            if (bpf_htons(PORT) == ITEM[i]) {                                  \
                *RET = 1;                                                      \
                break;                                                         \
            }                                                                  \
        }                                                                      \
    } while (0);

#define IS_EXCLUDE_IPRANGES(ITEM, IP, RET)                                     \
    do {                                                                       \
        *RET = 0;                                                              \
        for (int i = 0; i < MAX_ITEM_LEN && ITEM[i].net != 0; i++) {           \
            if (is_in_cidr(&ITEM[i], IP)) {                                    \
                *RET = 1;                                                      \
                break;                                                         \
            }                                                                  \
        }                                                                      \
    } while (0);

#define IS_INCLUDE_PORT(ITEM, PORT, RET)                                       \
    do {                                                                       \
        *RET = 0;                                                              \
        if (ITEM[0] != 0) {                                                    \
            for (int i = 0; i < MAX_ITEM_LEN && ITEM[i] != 0; i++) {           \
                if (bpf_htons(PORT) == ITEM[i]) {                              \
                    *RET = 1;                                                  \
                    break;                                                     \
                }                                                              \
            }                                                                  \
        } else {                                                               \
            *RET = 1;                                                          \
        }                                                                      \
    } while (0);

#define IS_INCLUDE_IPRANGES(ITEM, IP, RET)                                     \
    do {                                                                       \
        *RET = 0;                                                              \
        if (ITEM[0].net != 0) {                                                \
            for (int i = 0; i < MAX_ITEM_LEN && ITEM[i].net != 0; i++) {       \
                if (is_in_cidr(&ITEM[i], IP)) {                                \
                    *RET = 1;                                                  \
                    break;                                                     \
                }                                                              \
            }                                                                  \
        } else {                                                               \
            *RET = 1;                                                          \
        }                                                                      \
    } while (0);
