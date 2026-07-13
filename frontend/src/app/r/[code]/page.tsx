"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";

export default function RedirectPage() {
  const { code } = useParams<{ code: string }>();
  const [status, setStatus] = useState<string>("redirecting");
  const [targetUrl, setTargetUrl] = useState("");

  useEffect(() => {
    if (!code) return;

    // 同步方式直接重定向
    window.location.href = `http://localhost:8080/r/${code}`;
  }, [code]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="text-center">
        <div className="animate-spin text-4xl mb-4">⏳</div>
        <p className="text-gray-500">正在跳转...</p>
        <p className="text-xs text-gray-400 mt-2">短码: {code}</p>
      </div>
    </div>
  );
}
