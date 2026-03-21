/*
 * Copyright 2026 The Phantom Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

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
