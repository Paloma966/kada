"use client";

import { Search, Plus, SlidersHorizontal } from "lucide-react";
import Link from "next/link";

interface LinksToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  totalCount: number;
}

export function LinksToolbar({ search, onSearchChange, totalCount }: LinksToolbarProps) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      {/* Search */}
      <div className="relative flex-1 max-w-sm">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-gray-400" />
        <input
          type="text"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="搜索链接..."
          className="w-full rounded-lg border border-gray-200 bg-white py-2 pl-9 pr-4 text-sm placeholder:text-gray-400 focus:border-indigo-300 focus:outline-none focus:ring-2 focus:ring-indigo-100 transition"
        />
      </div>

      <div className="flex items-center gap-2">
        <span className="text-sm text-gray-400">{totalCount} 条链接</span>
        <Link
          href="/dashboard/links/new"
          className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition shadow-sm"
        >
          <Plus className="size-4" />
          创建
        </Link>
      </div>
    </div>
  );
}
