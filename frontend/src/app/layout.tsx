import type { Metadata } from "next";
import { Toaster } from "sonner";
import { I18nProvider } from "@/lib/i18n/context";
import { LocaleHtmlLang } from "@/components/LanguageSwitcher";
import "./globals.css";

// Server-rendered defaults use English; the client provider switches the visible
// copy to the persisted preference immediately after hydration.
export const metadata: Metadata = {
  title: "Kada - Short link management platform",
  description:
    "Smart short link management and analytics platform, compatible with WeChat/QQ/Xiaohongshu/SMS",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-gray-50">
        <I18nProvider>
          <LocaleHtmlLang />
          {children}
          <Toaster
            position="top-center"
            richColors
            closeButton
            toastOptions={{
              duration: 3000,
            }}
          />
        </I18nProvider>
      </body>
    </html>
  );
}
