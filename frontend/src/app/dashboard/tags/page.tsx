"use client";
import { Tag } from "lucide-react";
import { ComingSoon } from "@/components/ComingSoon";

export default function TagsPage() {
  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">标签</h1>
        <p className="text-sm text-gray-500 mt-1">用标签标记和筛选链接</p>
      </div>
      <ComingSoon
        icon={Tag}
        title="标签管理"
        description="即将支持标签功能，灵活标记和分类你的短链接"
      />
    </div>
  );
}
