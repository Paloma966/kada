"use client";
import { Users } from "lucide-react";
import { ComingSoon } from "@/components/ComingSoon";

export default function CustomersPage() {
  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">客户</h1>
        <p className="text-sm text-gray-500 mt-1">管理你的客户和设备指纹</p>
      </div>
      <ComingSoon
        icon={Users}
        title="客户管理"
        description="即将支持客户识别和设备指纹追踪，精准定位你的目标用户"
      />
    </div>
  );
}
