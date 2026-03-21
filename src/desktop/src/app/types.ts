export type DebugEventPayload = {
  timestamp_ns: number;
  session_id: string;
  event_type: number;
  event_type_name: string;
  pid: number;
  tgid: number;
  cpu: number;
  probe_id: string;
  payload_hex: string;
  payload_truncated: boolean;
  payload_utf8: string;
};

export type CpuJ = {
  label: string;
  user: number;
  nice: number;
  system: number;
  idle: number;
  iowait: number;
  irq: number;
  softirq: number;
  steal: number;
  guest: number;
  guest_nice: number;
};

export type NetDev = {
  name: string;
  rx_bytes: number;
  tx_bytes: number;
  rx_packets: number;
  tx_packets: number;
  rx_errors: number;
  tx_errors: number;
  rx_dropped: number;
  tx_dropped: number;
};

export type TaskRow = {
  tid: number;
  name: string;
  state: string;
  vm_peak_kb: number;
  vm_size_kb: number;
  vm_rss_kb: number;
  vm_hwm_kb: number;
  threads_count: number;
};
