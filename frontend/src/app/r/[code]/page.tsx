"use client";

import { useEffect } from "react";
import { useParams } from "next/navigation";
import { useT } from "@/lib/i18n";

export default function RedirectPage() {
  const { code } = useParams<{ code: string }>();
  const t = useT();

  useEffect(() => {
    if (!code) return;

    // Redirect synchronously
    window.location.href = `/r/${code}`;
  }, [code]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="text-center">
        <div className="animate-spin text-4xl mb-4">⏳</div>
        <p className="text-gray-500">{t("正在跳转...")}</p>
        <p className="text-xs text-gray-400 mt-2">{t("短码: {code}", { code })}</p>
      </div>
    </div>
  );
}
