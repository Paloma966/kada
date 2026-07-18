"use client";

import { useState } from "react";
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
  const [folderId, setFolderId] = useState(0);
  const [tagId, setTagId] = useState(0);
  const [sort, setSort] = useState("created_desc");

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

  return (
    <div className="space-y-5">
      <LinksToolbar
        search={search}
        onSearchChange={(v) => { setSearch(v); setPage(1); }}
        totalCount={totalCount}
        folders={folders}
        tags={tags}
        selectedFolderId={folderId}
        onFolderChange={(id) => { setFolderId(id); setPage(1); }}
        selectedTagId={tagId}
        onTagChange={(id) => { setTagId(id); setPage(1); }}
        sort={sort}
        onSortChange={(s) => { setSort(s); setPage(1); }}
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
          {links.map((link) => (
            <LinkCard key={link.id} link={link} onDelete={handleDelete} />
          ))}

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
