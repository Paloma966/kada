"use client";

import { Copy, ExternalLink, Pencil, Trash2, BarChart3, Check } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { toast } from "sonner";

export interface LinkItem {
  id: number;
  short_code: string;
  short_url: string;
  original_url: string;
  title: string;
  description?: string;
  click_count: number;
  is_active: boolean;
  created_at: string;
}

interface LinkCardProps {
  link: LinkItem;
  onDelete: (id: number) => void;
  workspaceSlug?: string;
}

export function LinkCard({ link, onDelete }: LinkCardProps) {
  const [copied, setCopied] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const handleCopy = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    navigator.clipboard.writeText(link.short_url);
    setCopied(true);
    toast.success("已复制到剪贴板");
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDelete = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (deleting) return;
    setDeleting(true);
    try {
      await onDelete(link.id);
      toast.success("链接已删除");
    } catch {
      toast.error("删除失败");
      setDeleting(false);
    }
  };

  return (
    <div className="group relative bg-white rounded-xl border border-gray-100 hover:border-gray-200 hover:shadow-md transition-all duration-200">
      <Link
        href={`/dashboard/links/${link.id}`}
        className="block p-4 sm:p-5"
      >
        <div className="flex items-start justify-between gap-4">
          {/* Left: Info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <h3 className="font-medium text-gray-900 truncate text-sm sm:text-base">
                {link.title || "未命名链接"}
              </h3>
              <span
                className={`shrink-0 px-1.5 py-0.5 text-[10px] font-medium rounded-full ${
                  link.is_active
                    ? "bg-green-100 text-green-700"
                    : "bg-gray-100 text-gray-500"
                }`}
              >
                {link.is_active ? "启用" : "停用"}
              </span>
            </div>

            <div className="flex items-center gap-2 text-sm">
              <span className="text-indigo-600 font-mono text-xs sm:text-sm truncate">
                {link.short_url}
              </span>
              <button
                onClick={handleCopy}
                className={`shrink-0 p-1 rounded transition-colors ${
                  copied
                    ? "text-green-500 bg-green-50"
                    : "text-gray-400 hover:text-indigo-500"
                }`}
                title="复制链接"
              >
                {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>

            <p className="text-xs text-gray-400 mt-1 truncate max-w-md">
              {link.original_url}
            </p>

            {link.description && (
              <p className="text-xs text-gray-500 mt-1 line-clamp-1">{link.description}</p>
            )}
          </div>

          {/* Right: Stats & Actions */}
          <div className="flex items-center gap-3 shrink-0">
            <div className="hidden sm:flex items-center gap-1 text-sm">
              <BarChart3 className="w-3.5 h-3.5 text-gray-400" />
              <span className="font-semibold text-gray-700 tabular-nums">
                {link.click_count.toLocaleString()}
              </span>
            </div>

            <a
              href={link.short_url}
              target="_blank"
              onClick={(e) => e.stopPropagation()}
              className="p-1.5 text-gray-400 hover:text-indigo-500 hover:bg-indigo-50 rounded-lg transition-colors"
              title="打开链接"
            >
              <ExternalLink className="w-4 h-4" />
            </a>

            <Link
              href={`/dashboard/links/${link.id}`}
              onClick={(e) => e.stopPropagation()}
              className="p-1.5 text-gray-400 hover:text-indigo-500 hover:bg-indigo-50 rounded-lg transition-colors"
              title="编辑"
            >
              <Pencil className="w-4 h-4" />
            </Link>

            <button
              onClick={handleDelete}
              disabled={deleting}
              className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors disabled:opacity-50"
              title="删除"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        </div>
      </Link>
    </div>
  );
}

/** Skeleton placeholder shown while loading */
export function LinkCardPlaceholder() {
  return (
    <div className="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
      <div className="flex items-start justify-between gap-4 animate-pulse">
        <div className="flex-1 space-y-2">
          <div className="flex items-center gap-2">
            <div className="h-5 w-32 bg-gray-200 rounded-md sm:w-44" />
            <div className="h-4 w-10 bg-gray-200 rounded-full" />
          </div>
          <div className="flex items-center gap-2">
            <div className="h-4 w-48 bg-gray-100 rounded-md" />
            <div className="h-3.5 w-3.5 bg-gray-100 rounded" />
          </div>
          <div className="h-3 w-64 bg-gray-100 rounded" />
        </div>
        <div className="flex items-center gap-3">
          <div className="hidden sm:block h-6 w-16 bg-gray-200 rounded-md" />
          <div className="h-8 w-8 bg-gray-100 rounded-lg" />
          <div className="h-8 w-8 bg-gray-100 rounded-lg" />
          <div className="h-8 w-8 bg-gray-100 rounded-lg" />
        </div>
      </div>
    </div>
  );
}
