"use client";
import { Folder } from "lucide-react";
import { ComingSoon } from "@/components/ComingSoon";

export default function FoldersPage() {
  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">文件夹</h1>
        <p className="text-sm text-gray-500 mt-1">用文件夹组织你的短链接</p>
      </div>
      <ComingSoon
        icon={Folder}
        title="文件夹管理"
        description="即将支持文件夹功能，帮你更好地组织和分类短链接"
      />
    </div>
  );
}
