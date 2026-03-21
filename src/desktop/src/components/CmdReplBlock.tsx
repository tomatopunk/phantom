import type { TFunction } from "i18next";

type Props = {
  t: TFunction;
  cmd: string;
  setCmd: (s: string) => void;
  cmdOut: string;
  runCmd: () => void;
  connected: boolean;
};

export function CmdReplBlock({ t, cmd, setCmd, cmdOut, runCmd, connected }: Props) {
  return (
    <div className="border-t border-shell-border p-2 space-y-1 shrink-0">
      <div className="text-xs text-shell-muted">{t("cmd.title")}</div>
      <div className="flex gap-1">
        <input
          className="flex-1 bg-black/40 border border-shell-border rounded px-1 font-mono-tight text-xs"
          value={cmd}
          onChange={(e) => setCmd(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && runCmd()}
        />
        <button
          type="button"
          disabled={!connected}
          className="px-2 text-xs rounded border border-shell-border disabled:opacity-40"
          onClick={runCmd}
        >
          {t("cmd.run")}
        </button>
      </div>
      <pre className="text-[10px] bg-black/30 rounded p-1 max-h-24 overflow-auto whitespace-pre-wrap">{cmdOut}</pre>
    </div>
  );
}
