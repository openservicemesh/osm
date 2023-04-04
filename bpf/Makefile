
CC=clang
CFLAGS=-O2 -g  -Wall -target bpf -I/usr/include/$(shell uname -m)-linux-gnu

BPF_FS=/sys/fs/bpf
PIN_OBJECT_NS_PATH=/sys/fs/bpf

MACROS:=
CNI_MODE ?= false
DEBUG ?= 1

# see https://stackoverflow.com/questions/15063298/how-to-check-kernel-version-in-makefile
KVER = $(shell uname -r)
KMAJ = $(shell echo $(KVER) | \
sed -e 's/^\([0-9][0-9]*\)\.[0-9][0-9]*\.[0-9][0-9]*.*/\1/')
KMIN = $(shell echo $(KVER) | \
sed -e 's/^[0-9][0-9]*\.\([0-9][0-9]*\)\.[0-9][0-9]*.*/\1/')
KREV = $(shell echo $(KVER) | \
sed -e 's/^[0-9][0-9]*\.[0-9][0-9]*\.\([0-9][0-9]*\).*/\1/')

kver_ge = $(shell \
echo test | awk '{if($(KMAJ) < $(1)) {print 0} else { \
if($(KMAJ) > $(1)) {print 1} else { \
if($(KMIN) < $(2)) {print 0} else { \
if($(KMIN) > $(2)) {print 1} else { \
if($(KREV) < $(3)) {print 0} else { print 1 } \
}}}}}' \
)

# See https://nakryiko.com/posts/bpf-tips-printk/, kernel will auto print newline if version greater than 5.9.0
ifneq ($(call kver_ge,5,8,999),1)
MACROS:= $(MACROS) -DPRINTNL # kernel version less
endif

ifeq ($(CNI_MODE),true)
	MACROS:= $(MACROS) -DENABLE_CNI_MODE=1
else
	MACROS:= $(MACROS) -DENABLE_CNI_MODE=0
endif

ifeq ($(DEBUG),1)
    MACROS:= $(MACROS) -DDEBUG
endif

CGROUP2_PATH ?= $(shell mount | grep cgroup2 | awk '{print $$3}' | grep -v "^/host" | head -n 1)
ifeq ($(CGROUP2_PATH),)
$(error It looks like your system does not have cgroupv2 enabled, or the automatic recognition fails. Please enable cgroupv2, or specify the path of cgroupv2 manually via CGROUP2_PATH parameter.)
endif

TARGETS=osm_cni_grp_connect.o osm_cni_sock_opt.o osm_cni_msg_redirect.o osm_cni_sock_ops.o osm_cni_tc_nat.o

$(BPF_FS):
	mountpoint -q $@ || mount -t bpf bpf $@

init-bpffs:
	mountpoint -q ${BPF_FS} || mount -t bpf bpf ${BPF_FS}

%.o: %.c
	$(CC) $(CFLAGS) $(MACROS) -c $< -o $@

generate-compilation-database:
	CC="$(CC)" CFLAGS="$(CFLAGS)" MACROS="$(MACROS)" scripts/generate-compilation-database.sh | tee compile_commands.json

compile: $(TARGETS)

compile-clean:
	[ -f osm_cni_grp_connect.c ] && sudo rm -f $(TARGETS)

# Map
load-map-osm_cki_fib:
	[ -f $(PIN_OBJECT_NS_PATH)/osm_cki_fib ] || sudo bpftool map create $(PIN_OBJECT_NS_PATH)/osm_cki_fib type lru_hash key 8 value 24 entries 65535 name osm_cki_fib

load-map-osm_pod_fib:
	[ -f $(PIN_OBJECT_NS_PATH)/osm_pod_fib ] || sudo bpftool map create $(PIN_OBJECT_NS_PATH)/osm_pod_fib type hash key 16 value 484 entries 1024 name osm_pod_fib

load-map-osm_proc_fib:
	[ -f $(PIN_OBJECT_NS_PATH)/osm_proc_fib ] || sudo bpftool map create $(PIN_OBJECT_NS_PATH)/osm_proc_fib type lru_hash key 4 value 4 entries 1024 name osm_proc_fib

load-map-osm_cgr_fib:
	[ -f $(PIN_OBJECT_NS_PATH)/osm_cgr_fib ] || sudo bpftool map create $(PIN_OBJECT_NS_PATH)/osm_cgr_fib type lru_hash key 8 value 32 entries 1024 name osm_cgr_fib

load-map-osm_mark_fib:
	[ -f $(PIN_OBJECT_NS_PATH)/osm_mark_fib ] || sudo bpftool map create $(PIN_OBJECT_NS_PATH)/osm_mark_fib type hash key 4 value 16 entries 65535 name osm_mark_fib

load-map-osm_nat_fib:
	[ -f $(PIN_OBJECT_NS_PATH)/osm_nat_fib ] || sudo bpftool map create $(PIN_OBJECT_NS_PATH)/osm_nat_fib type lru_hash key 36 value 24 entries 65535 name osm_nat_fib

load-map-osm_sock_fib:
	[ -f $(PIN_OBJECT_NS_PATH)/osm_sock_fib ] || sudo bpftool map create $(PIN_OBJECT_NS_PATH)/osm_sock_fib type sockhash key 36 value 4 entries 65535 name osm_sock_fib

clean-maps:
	sudo rm -f \
		$(PIN_OBJECT_NS_PATH)/osm_sock_fib \
		$(PIN_OBJECT_NS_PATH)/osm_nat_fib \
		$(PIN_OBJECT_NS_PATH)/osm_proc_fib \
		$(PIN_OBJECT_NS_PATH)/osm_cgr_fib \
		$(PIN_OBJECT_NS_PATH)/osm_pod_fib \
		$(PIN_OBJECT_NS_PATH)/osm_cki_fib \
		$(PIN_OBJECT_NS_PATH)/osm_mark_fib

load-osm_cni_sock_opt: load-map-osm_nat_fib
	sudo bpftool -m prog load osm_cni_sock_opt.o $(PIN_OBJECT_NS_PATH)/get_sockopts \
		map name osm_nat_fib pinned $(PIN_OBJECT_NS_PATH)/osm_nat_fib

attach-osm_cni_sock_opt:
	sudo bpftool cgroup attach $(CGROUP2_PATH) getsockopt pinned $(PIN_OBJECT_NS_PATH)/get_sockopts

clean-osm_cni_sock_opt:
	sudo bpftool cgroup detach $(CGROUP2_PATH) getsockopt pinned $(PIN_OBJECT_NS_PATH)/get_sockopts
	sudo rm $(PIN_OBJECT_NS_PATH)/get_sockopts

load-osm_cni_msg_redirect: load-map-osm_sock_fib
	sudo bpftool -m prog load osm_cni_msg_redirect.o $(PIN_OBJECT_NS_PATH)/redir \
		map name osm_sock_fib pinned $(PIN_OBJECT_NS_PATH)/osm_sock_fib

attach-osm_cni_msg_redirect:
	sudo bpftool prog attach pinned $(PIN_OBJECT_NS_PATH)/redir msg_verdict pinned $(PIN_OBJECT_NS_PATH)/osm_sock_fib

clean-osm_cni_msg_redirect:
	sudo bpftool prog detach pinned $(PIN_OBJECT_NS_PATH)/redir msg_verdict pinned $(PIN_OBJECT_NS_PATH)/osm_sock_fib
	sudo rm $(PIN_OBJECT_NS_PATH)/redir

load-osm_cni_grp_connect: load-map-osm_cki_fib load-map-osm_pod_fib load-map-osm_proc_fib load-map-osm_cgr_fib load-map-osm_mark_fib
	sudo bpftool -m prog loadall osm_cni_grp_connect.o $(PIN_OBJECT_NS_PATH)/connect \
		map name osm_cki_fib pinned $(PIN_OBJECT_NS_PATH)/osm_cki_fib \
		map name osm_pod_fib pinned $(PIN_OBJECT_NS_PATH)/osm_pod_fib \
		map name osm_mark_fib pinned $(PIN_OBJECT_NS_PATH)/osm_mark_fib \
		map name osm_proc_fib pinned $(PIN_OBJECT_NS_PATH)/osm_proc_fib \
		map name osm_cgr_fib pinned $(PIN_OBJECT_NS_PATH)/osm_cgr_fib

attach-osm_cni_grp_connect:
	sudo bpftool cgroup attach $(CGROUP2_PATH) connect4 pinned $(PIN_OBJECT_NS_PATH)/connect/cgroup_connect4

clean-osm_cni_grp_connect:
	sudo bpftool cgroup detach $(CGROUP2_PATH) connect4 pinned $(PIN_OBJECT_NS_PATH)/connect/cgroup_connect4
	sudo rm -rf $(PIN_OBJECT_NS_PATH)/connect

load-osm_cni_sock_ops: load-map-osm_cki_fib load-map-osm_proc_fib load-map-osm_nat_fib load-map-osm_sock_fib
	sudo bpftool -m prog load osm_cni_sock_ops.o $(PIN_OBJECT_NS_PATH)/sockops \
		map name osm_cki_fib pinned $(PIN_OBJECT_NS_PATH)/osm_cki_fib \
		map name osm_proc_fib pinned $(PIN_OBJECT_NS_PATH)/osm_proc_fib \
		map name osm_nat_fib pinned $(PIN_OBJECT_NS_PATH)/osm_nat_fib \
		map name osm_sock_fib pinned $(PIN_OBJECT_NS_PATH)/osm_sock_fib

attach-osm_cni_sock_ops:
	sudo bpftool cgroup attach $(CGROUP2_PATH) sock_ops pinned $(PIN_OBJECT_NS_PATH)/sockops

clean-osm_cni_sock_ops:
	sudo bpftool cgroup detach $(CGROUP2_PATH) sock_ops pinned $(PIN_OBJECT_NS_PATH)/sockops
	sudo rm -rf $(PIN_OBJECT_NS_PATH)/sockops

load: compile load-from-obj

load-from-obj: init-bpffs load-osm_cni_grp_connect load-osm_cni_sock_ops load-osm_cni_sock_opt load-osm_cni_msg_redirect

attach: attach-osm_cni_grp_connect attach-osm_cni_sock_ops attach-osm_cni_sock_opt attach-osm_cni_msg_redirect

clean: clean-osm_cni_grp_connect clean-osm_cni_sock_ops clean-osm_cni_sock_opt clean-osm_cni_msg_redirect clean-maps compile-clean