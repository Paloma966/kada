"use client";

import { Search, Plus, Folder, Tag } from "lucide-react";
import Link from "next/link";

export interface FilterOption {
  id: number;
  name: string;
  color?: string;
}

interface LinksToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  totalCount: number;
  folders: FilterOption[];
  tags: FilterOption[];
  selectedFolderId: number;
  onFolderChange: (id: number) => void;
  selectedTagId: number;
  onTagChange: (id: number) => void;
}

export function LinksToolbar({
  search,
  onSearchChange,
  totalCount,
  folders,
  tags,
  selectedFolderId,
  onFolderChange,
  selectedTagId,
  onTagChange,
}: LinksToolbarProps) {
  const hasFilters = folders.length > 0 || tags.length > 0;

  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-2 flex-1">
        {/* Search */}
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-gray-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="搜索链接..."
            className="w-full rounded-lg border border-gray-200 bg-white py-2 pl-9 pr-4 text-sm placeholder:text-gray-400 focus:border-indigo-300 focus:outline-none focus:ring-2 focus:ring-indigo-100 transition"
          />
        </div>

        {/* Folder filter */}
        {folders.length > 0 && (
          <div className="relative">
            <select
              value={selectedFolderId}
              onChange={(e) => onFolderChange(Number(e.target.value))}
              className="appearance-none rounded-lg border border-gray-200 bg-white py-2 pl-8 pr-8 text-sm text-gray-600 focus:border-indigo-300 focus:outline-none focus:ring-2 focus:ring-indigo-100 transition cursor-pointer"
            >
              <option value={0}>全部文件夹</option>
              {folders.map((f) => (
                <option key={f.id} value={f.id}>{f.name}</option>
              ))}
            </select>
            <Folder className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-gray-400 pointer-events-none" />
          </div>
        )}

        {/* Tag filter */}
        {tags.length > 0 && (
          <div className="relative">
            <select
              value={selectedTagId}
              onChange={(e) => onTagChange(Number(e.target.value))}
              className="appearance-none rounded-lg border border-gray-200 bg-white py-2 pl-8 pr-8 text-sm text-gray-600 focus:border-indigo-300 focus:outline-none focus:ring-2 focus:ring-indigo-100 transition cursor-pointer"
            >
              <option value={0}>全部标签</option>
              {tags.map((t) => (
                <option key={t.id} value={t.id}>{t.name}</option>
              ))}
            </select>
            <Tag className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-gray-400 pointer-events-none" />
          </div>
        )}
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
