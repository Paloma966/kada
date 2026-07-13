"use client";
import { Globe } from "lucide-react";
import { ComingSoon } from "@/components/ComingSoon";

export default function DomainsPage() {
  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">域名</h1>
        <p className="text-sm text-gray-500 mt-1">管理你的自定义域名</p>
      </div>
      <ComingSoon
        icon={Globe}
        title="自定义域名管理"
        description="即将支持绑定自定义域名，让你的短链接更具品牌辨识度"
      />
    </div>
  );
}
