"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Copy, ExternalLink } from "lucide-react";
import { linksAPI } from "@/lib/api";
import { getToken } from "@/lib/auth";

interface LinkDetail {
  id: number;
  short_code: string;
  short_url: string;
  original_url: string;
  title: string;
  description: string;
  domain: string;
  click_count: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export default function LinkDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [link, setLink] = useState<LinkDetail | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const token = getToken();
    if (!token) return;
    linksAPI.get(token, id).then(setLink).catch(console.error).finally(() => setLoading(false));
  }, [id]);

  if (loading) return <div className="text-center py-12 text-gray-400">加载中...</div>;
  if (!link) return <div className="text-center py-12 text-red-500">链接不存在</div>;

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  return (
    <div className="max-w-2xl mx-auto">
      <Link
        href="/dashboard"
        className="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-indigo-600 mb-6 transition"
      >
        <ArrowLeft className="w-4 h-4" /> 返回
      </Link>

      <div className="bg-white rounded-xl p-6 shadow-sm">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-xl font-bold text-gray-900">
            {link.title || link.short_code}
          </h1>
          <span className={`px-2 py-1 text-xs rounded-full ${link.is_active ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>
            {link.is_active ? "启用" : "停用"}
          </span>
        </div>

        {/* Short URL */}
        <div className="mb-6 p-4 bg-indigo-50 rounded-lg">
          <div className="flex items-center gap-2">
            <span className="text-indigo-600 font-mono font-medium text-lg">{link.short_url}</span>
            <button
              onClick={() => copyToClipboard(link.short_url)}
              className="ml-auto p-2 text-indigo-400 hover:text-indigo-600 transition"
            >
              <Copy className="w-4 h-4" />
            </button>
            <a
              href={link.short_url}
              target="_blank"
              className="p-2 text-indigo-400 hover:text-indigo-600 transition"
            >
              <ExternalLink className="w-4 h-4" />
            </a>
          </div>
          <p className="text-xs text-indigo-400 mt-1 truncate">→ {link.original_url}</p>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-2 gap-4 mb-6">
          <div className="p-4 bg-gray-50 rounded-lg text-center">
            <div className="text-2xl font-bold text-gray-900">{link.click_count.toLocaleString()}</div>
            <div className="text-sm text-gray-500">总点击</div>
          </div>
          <div className="p-4 bg-gray-50 rounded-lg text-center">
            <div className="text-2xl font-bold text-gray-900">
              {new Date(link.created_at).toLocaleDateString("zh-CN")}
            </div>
            <div className="text-sm text-gray-500">创建日期</div>
          </div>
        </div>

        {/* Details */}
        <div className="space-y-3 text-sm">
          <div className="flex justify-between py-2 border-b">
            <span className="text-gray-500">短码</span>
            <span className="text-gray-900 font-mono">{link.short_code}</span>
          </div>
          <div className="flex justify-between py-2 border-b">
            <span className="text-gray-500">域名</span>
            <span className="text-gray-900">{link.domain}</span>
          </div>
          {link.description && (
            <div className="flex justify-between py-2 border-b">
              <span className="text-gray-500">描述</span>
              <span className="text-gray-900">{link.description}</span>
            </div>
          )}
          <div className="flex justify-between py-2 border-b">
            <span className="text-gray-500">创建时间</span>
            <span className="text-gray-900">{new Date(link.created_at).toLocaleString("zh-CN")}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
