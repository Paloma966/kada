"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Copy, ExternalLink, Pencil, Save, X, ToggleLeft, ToggleRight, Check } from "lucide-react";
import { toast } from "sonner";
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
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState(false);
  const [copied, setCopied] = useState(false);

  // Edit form state
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");

  const fetchLink = () => {
    const token = getToken();
    if (!token) return;
    setLoading(true);
    linksAPI.get(token, id)
      .then((data) => {
        setLink(data);
        setEditTitle(data.title || "");
        setEditDescription(data.description || "");
      })
      .catch((e) => toast.error("加载链接失败"))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchLink(); }, [id]);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    toast.success("已复制到剪贴板");
    setTimeout(() => setCopied(false), 2000);
  };

  const handleSave = async () => {
    const token = getToken();
    if (!token || !link) return;
    setSaving(true);
    try {
      const updated = await linksAPI.update(token, link.id, {
        title: editTitle,
        description: editDescription,
      });
      setLink({ ...link, ...updated });
      setEditing(false);
      toast.success("链接已更新");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "更新失败");
    } finally {
      setSaving(false);
    }
  };

  const handleToggleActive = async () => {
    const token = getToken();
    if (!token || !link) return;
    try {
      const updated = await linksAPI.update(token, link.id, {
        is_active: !link.is_active,
      });
      setLink({ ...link, is_active: !link.is_active });
      toast.success(updated.is_active ? "链接已启用" : "链接已停用");
    } catch (e: unknown) {
      toast.error("状态更新失败");
    }
  };

  const handleCancel = () => {
    setEditTitle(link?.title || "");
    setEditDescription(link?.description || "");
    setEditing(false);
  };

  if (loading) {
    return (
      <div className="max-w-2xl mx-auto">
        <div className="animate-pulse space-y-4">
          <div className="h-6 w-24 bg-gray-200 rounded" />
          <div className="h-64 bg-gray-100 rounded-xl" />
        </div>
      </div>
    );
  }

  if (!link) {
    return (
      <div className="max-w-2xl mx-auto text-center py-16">
        <div className="text-4xl mb-4">🔍</div>
        <h2 className="text-lg font-medium text-gray-900">链接不存在</h2>
        <p className="text-sm text-gray-500 mt-1">该链接可能已被删除</p>
        <Link href="/dashboard" className="inline-block mt-4 text-indigo-600 hover:text-indigo-500 text-sm">
          返回列表
        </Link>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto">
      {/* Back */}
      <Link
        href="/dashboard"
        className="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-indigo-600 mb-6 transition"
      >
        <ArrowLeft className="w-4 h-4" /> 返回
      </Link>

      <div className="bg-white rounded-xl p-6 shadow-sm">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          {editing ? (
            <input
              type="text"
              value={editTitle}
              onChange={(e) => setEditTitle(e.target.value)}
              className="flex-1 text-xl font-bold text-gray-900 border-b-2 border-indigo-300 focus:border-indigo-500 focus:outline-none px-2 py-1 mr-4"
              placeholder="链接标题"
              autoFocus
            />
          ) : (
            <h1 className="text-xl font-bold text-gray-900">{link.title || "未命名链接"}</h1>
          )}
          <div className="flex items-center gap-2">
            {/* Active toggle */}
            <button
              onClick={handleToggleActive}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-full font-medium transition ${
                link.is_active
                  ? "bg-green-100 text-green-700 hover:bg-green-200"
                  : "bg-gray-100 text-gray-500 hover:bg-gray-200"
              }`}
            >
              {link.is_active ? (
                <ToggleRight className="w-3.5 h-3.5" />
              ) : (
                <ToggleLeft className="w-3.5 h-3.5" />
              )}
              {link.is_active ? "启用中" : "已停用"}
            </button>
            {/* Edit toggle */}
            {!editing && (
              <button
                onClick={() => setEditing(true)}
                className="p-2 text-gray-400 hover:text-indigo-600 transition"
                title="编辑"
              >
                <Pencil className="w-4 h-4" />
              </button>
            )}
          </div>
        </div>

        {/* Short URL */}
        <div className="mb-6 p-4 bg-indigo-50 rounded-lg">
          <div className="flex items-center gap-2">
            <span className="text-indigo-600 font-mono font-medium text-lg">{link.short_url}</span>
            <button
              onClick={() => copyToClipboard(link.short_url)}
              className={`ml-auto p-2 transition rounded-lg ${
                copied
                  ? "text-green-500 bg-green-50"
                  : "text-indigo-400 hover:text-indigo-600"
              }`}
              title="复制链接"
            >
              {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
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

        {/* Edit Form */}
        {editing && (
          <div className="mb-6 p-4 border-2 border-indigo-200 rounded-lg bg-indigo-50/50">
            <h3 className="font-medium text-sm text-indigo-700 mb-3">编辑链接</h3>
            <div className="space-y-3">
              <div>
                <label className="block text-xs text-gray-500 mb-1">描述</label>
                <textarea
                  value={editDescription}
                  onChange={(e) => setEditDescription(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  placeholder="简短描述"
                  rows={3}
                />
              </div>
              <div className="flex gap-2 justify-end">
                <button
                  onClick={handleCancel}
                  disabled={saving}
                  className="inline-flex items-center gap-1 px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg transition"
                >
                  <X className="w-4 h-4" /> 取消
                </button>
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="inline-flex items-center gap-1 px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg transition disabled:opacity-50"
                >
                  <Save className="w-4 h-4" /> {saving ? "保存中..." : "保存"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className="p-4 bg-gray-50 rounded-lg text-center">
            <div className="text-2xl font-bold text-gray-900">{link.click_count.toLocaleString()}</div>
            <div className="text-xs text-gray-500 mt-0.5">总点击</div>
          </div>
          <div className="p-4 bg-gray-50 rounded-lg text-center">
            <div className="text-sm font-bold text-gray-900">
              {new Date(link.created_at).toLocaleDateString("zh-CN")}
            </div>
            <div className="text-xs text-gray-500 mt-0.5">创建日期</div>
          </div>
          <div className="p-4 bg-gray-50 rounded-lg text-center">
            <div className="text-sm font-bold text-gray-900">
              {new Date(link.updated_at).toLocaleDateString("zh-CN")}
            </div>
            <div className="text-xs text-gray-500 mt-0.5">最近更新</div>
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
              <span className="text-gray-900 max-w-[60%] text-right">{link.description}</span>
            </div>
          )}
          <div className="flex justify-between py-2 border-b">
            <span className="text-gray-500">创建时间</span>
            <span className="text-gray-900">{new Date(link.created_at).toLocaleString("zh-CN")}</span>
          </div>
          <div className="flex justify-between py-2">
            <span className="text-gray-500">目标 URL</span>
            <a href={link.original_url} target="_blank" className="text-indigo-600 hover:underline max-w-[60%] text-right truncate">
              {link.original_url}
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
