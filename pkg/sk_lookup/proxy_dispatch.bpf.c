#include <linux/bpf.h>
#include <linux/types.h>

#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

#define MAX_SOCKETS (1)
#define MAX_DESTINATIONS (64)

/* destinations */
struct
{
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(key_size, sizeof(__u16));
	__uint(value_size, sizeof(__u8));
	__uint(max_entries, MAX_DESTINATIONS);
} destinations SEC(".maps");

/* sockets */
struct
{
	__uint(type, BPF_MAP_TYPE_SOCKMAP);
	__uint(key_size, sizeof(__u32));
	__uint(value_size, sizeof(__u64));
	__uint(max_entries, MAX_SOCKETS);
} sockets SEC(".maps");

/* Dispatcher program */
SEC("sk_lookup/dispatch")
int dispatch(struct bpf_sk_lookup *ctx)
{
	const __u32 zero = 0;
	struct bpf_sock *sk;
	__u16 port;
	__u8 *exists;
	long err;

	/* Are we supposed to listeing on this port? */
	port = ctx->local_port;
	exists = bpf_map_lookup_elem(&destinations, &port);
	if (!exists)
		return SK_PASS;

	/* Get TCP Proxy socket */
	sk = bpf_map_lookup_elem(&sockets, &zero);
	if (!sk)
		return SK_DROP;

	/* Dispatch the packet to the socket */
	err = bpf_sk_assign(ctx, sk, 0);
	bpf_sk_release(sk);
	return err ? SK_DROP : SK_PASS;
}
