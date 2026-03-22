/**
 * Props for code-like inputs: disable macOS/WebKit auto-capitalization, spell check, etc.
 */
export const technicalInputProps = {
  autoCapitalize: "none" as const,
  autoCorrect: "off" as const,
  spellCheck: false,
  autoComplete: "off" as const,
};
