import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";

import en from "./locales/en.json";
import zh from "./locales/zh.json";

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      zh: { translation: zh },
    },
    supportedLngs: ["en", "zh"],
    fallbackLng: "zh",
    interpolation: { escapeValue: false },
    detection: {
      order: ["localStorage", "navigator"],
      caches: ["localStorage"],
      lookupLocalStorage: "phantom-desktop-lang",
    },
  });

function syncHtmlLang(lng: string) {
  document.documentElement.lang = lng === "en" ? "en" : "zh-CN";
}

syncHtmlLang(i18n.resolvedLanguage ?? i18n.language);
i18n.on("languageChanged", syncHtmlLang);

export default i18n;
