/* Minimal struct sock / sock_common for BPF CO-RE field paths.
 * Layout is not authoritative; the loader relocates using kernel BTF.
 */
#ifndef PHANTOM_SOCK_H
#define PHANTOM_SOCK_H

#ifndef __be32
typedef __u32 __be32;
#endif

struct sock_common {
	__u32 skc_addrpair;
	__u32 skc_hash;
	__u16 skc_dport;
	__u16 skc_num;
	__be32 skc_daddr;
	__be32 skc_rcv_saddr;
};

struct sock {
	struct sock_common __sk_common;
};

#endif /* PHANTOM_SOCK_H */
