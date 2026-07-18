"use client";

import { useState, useEffect, useRef } from "react";
import Link from "next/link";
import { Plus, Link2 } from "lucide-react";
import useSWR from "swr";
import { toast } from "sonner";
import { linksAPI, foldersAPI, tagsAPI } from "@/lib/api";
import { getToken } from "@/lib/auth";
import { LinkCard, LinkCardPlaceholder } from "@/components/LinkCard";
import { LinksToolbar } from "@/components/LinksToolbar";
import type { LinkItem } from "@/components/LinkCard";

const PAGE_SIZE = 20;

export default function DashboardPage() {
  const token = getToken();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const searchTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const [folderId, setFolderId] = useState(0);
  const [tagId, setTagId] = useState(0);
  const [sort, setSort] = useState("created_desc");
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [batchTagId, setBatchTagId] = useState<number>(0);

  // 搜索防抖 300ms
  useEffect(() => {
    clearTimeout(searchTimer.current);
    searchTimer.current = setTimeout(() => {
      setSearch(searchInput);
      setPage(1);
    }, 300);
    return () => clearTimeout(searchTimer.current);
  }, [searchInput]);

  const { data, error, isLoading, mutate } = useSWR(
    token ? [`links`, page, search, folderId, tagId, sort] : null,
    () => linksAPI.list(token!, page, PAGE_SIZE, search, folderId, tagId, sort)
  );

  const { data: folderData } = useSWR(
    token ? "folders" : null,
    () => foldersAPI.list(token!)
  );

  const { data: tagData } = useSWR(
    token ? "tags" : null,
    () => tagsAPI.list(token!)
  );

  const links: LinkItem[] = data?.links ?? [];
  const totalCount: number = data?.total_count ?? 0;
  const totalPages = Math.ceil(totalCount / PAGE_SIZE);
  const folders = folderData?.folders ?? [];
  const tags = tagData?.tags ?? [];

  const handleDelete = async (id: number) => {
    if (!token) return;
    try {
      await linksAPI.delete(token, id);
      mutate();
    } catch {
      toast.error("删除失败");
      throw new Error("Delete failed");
    }
  };

  const handleSelect = (id: number, checked: boolean) => {
    setSelectedIds(prev => {
      const next = new Set(prev);
      checked ? next.add(id) : next.delete(id);
      return next;
    });
  };

  const handleSelectAll = () => {
    if (selectedIds.size === links.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(links.map(l => l.id)));
    }
  };

  const handleBatchDelete = async () => {
    if (!token || selectedIds.size === 0) return;
    try {
      await linksAPI.batchDelete(token, Array.from(selectedIds));
      toast.success(`已删除 ${selectedIds.size} 条链接`);
      setSelectedIds(new Set());
      mutate();
    } catch {
      toast.error("批量删除失败");
    }
  };

  const handleExport = async () => {
    if (!token) return;
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
      const res = await fetch(`${apiUrl}/api/links/export`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url; a.download = "kada-links.csv"; a.click();
      URL.revokeObjectURL(url);
      toast.success("导出成功");
    } catch {
      toast.error("导出失败");
    }
  };

  const handleBatchTag = async () => {
    if (!token || selectedIds.size === 0 || batchTagId === 0) return;
    try {
      await linksAPI.batchTag(token, Array.from(selectedIds), batchTagId);
      toast.success(`已为 ${selectedIds.size} 条链接添加标签`);
      setBatchTagId(0);
      mutate();
    } catch {
      toast.error("批量打标签失败");
    }
  };

  return (
    <div className="space-y-5">
      <LinksToolbar
        search={searchInput}
        onSearchChange={(v) => setSearchInput(v)}
        totalCount={totalCount}
        folders={folders}
        tags={tags}
        selectedFolderId={folderId}
        onFolderChange={(id) => { setFolderId(id); setPage(1); }}
        selectedTagId={tagId}
        onTagChange={(id) => { setTagId(id); setPage(1); }}
        sort={sort}
        onSortChange={(s) => { setSort(s); setPage(1); }}
        onExport={handleExport}
      />

      {error ? (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="flex size-16 items-center justify-center rounded-2xl bg-red-50 mb-4">
            <div className="text-3xl">😞</div>
          </div>
          <h3 className="text-lg font-semibold text-gray-900">加载失败</h3>
          <p className="mt-1 text-sm text-gray-500">请检查网络后重试</p>
          <button
            onClick={() => mutate()}
            className="mt-4 text-sm font-medium text-indigo-600 hover:text-indigo-500"
          >
            重新加载
          </button>
        </div>
      ) : isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <LinkCardPlaceholder key={i} />
          ))}
        </div>
      ) : links.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="flex size-16 items-center justify-center rounded-2xl bg-gray-100 mb-4">
            <Link2 className="size-8 text-gray-400" />
          </div>
          <h3 className="text-lg font-semibold text-gray-900">
            {search ? "没有匹配的链接" : "创建你的第一个短链接"}
          </h3>
          <p className="mt-1 text-sm text-gray-500 max-w-sm">
            {search
              ? "换个关键词试试"
              : "缩短、分享并追踪你的链接，兼容微信、QQ、小红书等平台"}
          </p>
          {!search && (
            <Link
              href="/dashboard/links/new"
              className="mt-5 inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-indigo-700 transition shadow-sm"
            >
              <Plus className="size-4" />
              创建链接
            </Link>
          )}
        </div>
      ) : (
        <div className="space-y-2">
          {/* Select all bar */}
          <div className="flex items-center gap-2 px-1">
            <label className="flex items-center gap-2 text-sm text-gray-500 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={links.length > 0 && selectedIds.size === links.length}
                onChange={handleSelectAll}
                className="size-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer"
              />
              全选
            </label>
            {selectedIds.size > 0 && (
              <span className="text-xs text-gray-400">已选 {selectedIds.size} 条</span>
            )}
          </div>

          {links.map((link) => (
            <LinkCard
              key={link.id}
              link={link}
              onDelete={handleDelete}
              selectable
              selected={selectedIds.has(link.id)}
              onSelect={handleSelect}
            />
          ))}

          {/* Batch action bar */}
          {selectedIds.size > 0 && (
            <div className="sticky bottom-0 flex items-center justify-between gap-3 rounded-xl bg-white border border-gray-200 shadow-lg p-4 mt-4">
              <span className="text-sm font-medium text-gray-700">已选 {selectedIds.size} 条链接</span>
              <div className="flex items-center gap-2">
                <select
                  value={batchTagId}
                  onChange={(e) => setBatchTagId(Number(e.target.value))}
                  className="rounded-lg border border-gray-200 px-3 py-1.5 text-sm text-gray-600 focus:border-indigo-300 focus:outline-none bg-white"
                >
                  <option value={0}>添加标签...</option>
                  {tags.map((t: { id: number; name: string }) => (
                    <option key={t.id} value={t.id}>{t.name}</option>
                  ))}
                </select>
                <button onClick={handleBatchTag} disabled={batchTagId === 0}
                  className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed transition">
                  批量打标签
                </button>
                <button onClick={handleBatchDelete}
                  className="rounded-lg bg-red-50 px-3 py-1.5 text-sm font-medium text-red-600 hover:bg-red-100 transition">
                  批量删除
                </button>
                <button onClick={() => setSelectedIds(new Set())}
                  className="rounded-lg border border-gray-200 px-3 py-1.5 text-sm text-gray-500 hover:bg-gray-50 transition">
                  取消选择
                </button>
              </div>
            </div>
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-1 pt-6 pb-4">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-3 py-1.5 text-sm rounded-lg border border-gray-200 bg-white hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
              >
                上一页
              </button>

              {generatePageNumbers(page, totalPages).map((p, i) =>
                p === null ? (
                  <span key={`dot-${i}`} className="px-1 text-gray-400 text-sm">...</span>
                ) : (
                  <button
                    key={p}
                    onClick={() => setPage(p)}
                    className={`min-w-[2rem] h-8 text-sm rounded-lg transition ${
                      p === page
                        ? "bg-indigo-600 text-white font-medium shadow-sm"
                        : "border border-gray-200 bg-white hover:bg-gray-50 text-gray-600"
                    }`}
                  >
                    {p}
                  </button>
                )
              )}

              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={page >= totalPages}
                className="px-3 py-1.5 text-sm rounded-lg border border-gray-200 bg-white hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
              >
                下一页
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function generatePageNumbers(current: number, total: number): (number | null)[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
  const pages: (number | null)[] = [1];
  if (current > 3) pages.push(null);
  for (let i = Math.max(2, current - 1); i <= Math.min(total - 1, current + 1); i++) {
    pages.push(i);
  }
  if (current < total - 2) pages.push(null);
  pages.push(total);
  return pages;
}
