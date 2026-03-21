import type { TFunction } from "i18next";

type Props = {
  t: TFunction;
  sessionId: string | null;
  connected: boolean;
  capturing: boolean;
  metricsAt: string | null;
};

export function AppFooter({ t, sessionId, connected, capturing, metricsAt }: Props) {
  return (
    <footer className="shrink-0 px-2 py-1 border-t border-shell-border text-[11px] text-shell-muted flex flex-wrap gap-3 bg-shell-panel">
      <span>
        {t("footer.session")} {sessionId ?? t("common.dash")}
      </span>
      <span>
        {t("footer.connected")} {connected ? t("common.yes") : t("common.no")}
      </span>
      <span>
        {t("footer.capture")} {capturing ? t("footer.capturing") : t("footer.stopped")}
      </span>
      <span>
        {t("footer.metricsRefresh")} {metricsAt ?? t("common.dash")}
      </span>
      <span className="ml-auto">{t("footer.shortcutExport")}</span>
    </footer>
  );
}
