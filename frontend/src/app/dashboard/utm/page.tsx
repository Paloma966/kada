"use client";
import { ArrowRightLeft } from "lucide-react";
import { ComingSoon } from "@/components/ComingSoon";

export default function UTMPage() {
  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">UTM 模板</h1>
        <p className="text-sm text-gray-500 mt-1">管理 UTM 参数模板</p>
      </div>
      <ComingSoon
        icon={ArrowRightLeft}
        title="UTM 模板"
        description="即将支持 UTM 参数模板，轻松追踪营销活动效果"
      />
    </div>
  );
}
