import raw from "./probes.json";

export type PresetMode = "full_c" | "template_sec" | "template_code";

export type ProbePreset = {
  id: string;
  name: string;
  description: string;
  mode: PresetMode;
  attach: string;
  sec: string | null;
  cTemplate: string | null;
};

export const probePresets = raw as ProbePreset[];
