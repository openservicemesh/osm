#pragma once
#define SOCK_IP_MARK_PORT 15050

#ifndef OUT_REDIRECT_PORT
#define OUT_REDIRECT_PORT 15001
#endif

#ifndef IN_REDIRECT_PORT
#define IN_REDIRECT_PORT 15003
#endif

#ifndef SIDECAR_USER_ID
#define SIDECAR_USER_ID 1500
#endif

#ifndef DNS_CAPTURE_PORT
#define DNS_CAPTURE_PORT 15053
#endif

// 127.0.0.6 (network order)
static const __u32 sidecar_ip = 127 + (6 << 24);
// ::6 (network order)
static const __u32 sidecar_ip6[4] = {0, 0, 0, 6 << 24};
