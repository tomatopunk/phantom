import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

export function GlossaryTip({
  term,
  labelKey,
  label,
  children,
}: {
  term: string;
  /** i18n key under translation root (e.g. metrics.load1m) */
  labelKey?: string;
  /** Fallback if labelKey omitted */
  label?: string;
  children?: ReactNode;
}) {
  const { t } = useTranslation();
  const title = t(`glossary.${term}`, { defaultValue: term });
  const text = children ?? (labelKey ? t(labelKey) : label ?? term);
  return (
    <span className="inline-flex items-center gap-0.5 border-b border-dotted border-shell-muted cursor-help" title={title}>
      {text}
      <span className="text-shell-muted text-[10px]">?</span>
    </span>
  );
}
