"use client";

import { use, useEffect, useState, useRef } from "react";
import Link from "next/link";
import {
  Copy, ExternalLink, Pencil, Save, X, Check, QrCode, Download,
  BarChart3, Globe, Clock, Shield, Smartphone, Tags, ArrowLeft, Folder,
} from "lucide-react";
import { toast } from "sonner";
import useSWR from "swr";
import { linksAPI, foldersAPI, tagsAPI } from "@/lib/api";
import { getToken } from "@/lib/auth";

interface TagInfo {
  id: number;
  name: string;
  color: string;
}

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
  utm_source?: string;
  utm_medium?: string;
  utm_campaign?: string;
  utm_term?: string;
  utm_content?: string;
  ios_url?: string;
  android_url?: string;
  password_hash?: string;
  expires_at?: string;
  folder_id?: number;
  folder_name?: string;
  tags?: TagInfo[];
}

const platformLabels: Record<string, string> = {
  browser: "浏览器", wechat: "微信", qq: "QQ",
  weibo: "微博", xiaohongshu: "小红书", sms: "短信", unknown: "其他",
};

export default function LinkDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);

  const [link, setLink] = useState<LinkDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState(false);
  const [copied, setCopied] = useState(false);

  // Edit form
  const [editTitle, setEditTitle] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [editOriginalUrl, setEditOriginalUrl] = useState("");
  const [editPassword, setEditPassword] = useState("");
  const [editExpiresAt, setEditExpiresAt] = useState("");
  const [editFolderId, setEditFolderId] = useState<number | null>(null);
  const [editTagIds, setEditTagIds] = useState<number[]>([]);

  // QR code
  const [showQR, setShowQR] = useState(false);
  const [qrURL, setQrURL] = useState<string | null>(null);

  // Analytics
  const [showAnalytics, setShowAnalytics] = useState(false);
  const [clickData, setClickData] = useState<{ daily: { date: string; count: number }[]; platforms: { platform: string; count: number }[] } | null>(null);

  const token = getToken();
  const { data: folderData } = useSWR(token ? "folders" : null, () => foldersAPI.list(token!));
  const { data: tagData } = useSWR(token ? "tags" : null, () => tagsAPI.list(token!));
  const folders = folderData?.folders ?? [];
  const allTags = tagData?.tags ?? [];

  const fetchLink = () => {
    if (!token) return;
    setLoading(true);
    linksAPI.get(token, id)
      .then((data) => {
        const l = data.link || data;
        setLink(l);
        setEditTitle(l.title || "");
        setEditDescription(l.description || "");
        setEditOriginalUrl(l.original_url || "");
        setEditExpiresAt(l.expires_at ? l.expires_at.slice(0, 16) : "");
        setEditFolderId(l.folder_id ?? null);
        setEditTagIds((l.tags ?? []).map((t: TagInfo) => t.id));
      })
      .catch(() => toast.error("加载链接失败"))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchLink(); }, [id]);

  // Fetch per-link analytics
  useEffect(() => {
    if (showAnalytics && link && token) {
      Promise.all([
        fetch(`${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/api/analytics/daily?link_id=${link.id}`, {
          headers: { Authorization: `Bearer ${token}` },
        }).then(r => r.json()),
        fetch(`${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/api/analytics/platforms?link_id=${link.id}`, {
          headers: { Authorization: `Bearer ${token}` },
        }).then(r => r.json()),
      ]).then(([dailyData, platformsData]) => {
        setClickData({
          daily: dailyData.daily ?? [],
          platforms: platformsData.platforms ?? [],
        });
      }).catch(() => toast.error("加载统计数据失败"));
    }
  }, [showAnalytics, link, token]);

  // QR code
  useEffect(() => {
    if (showQR && link && !qrURL) {
      import("qrcode").then((QRCode) => {
        const canvas = document.createElement("canvas");
        QRCode.toCanvas(canvas, link.short_url, {
          width: 256, margin: 2,
          color: { dark: "#000000", light: "#ffffff" },
          errorCorrectionLevel: "M",
        });
        setQrURL(canvas.toDataURL("image/png"));
      });
    }
    if (!showQR) setQrURL(null);
  }, [showQR, link, qrURL]);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    toast.success("已复制");
    setTimeout(() => setCopied(false), 2000);
  };

  const handleSave = async () => {
    if (!token || !link) return;
    setSaving(true);
    try {
      const payload: Record<string, unknown> = {
        title: editTitle,
        description: editDescription,
        original_url: editOriginalUrl,
        folder_id: editFolderId,
        tag_ids: editTagIds,
      };
      if (editPassword) payload.password = editPassword;
      if (editExpiresAt) payload.expires_at = new Date(editExpiresAt).toISOString();
      const updated = await linksAPI.update(token, link.id, payload);
      setLink({ ...link, ...updated });
      setEditing(false);
      toast.success("已更新");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "更新失败");
    } finally {
      setSaving(false);
    }
  };

  const handleToggleActive = async () => {
    if (!token || !link) return;
    try {
      await linksAPI.update(token, link.id, { is_active: !link.is_active });
      setLink({ ...link, is_active: !link.is_active });
      toast.success(link.is_active ? "已停用" : "已启用");
    } catch {
      toast.error("状态更新失败");
    }
  };

  const toggleEditTag = (tagId: number) => {
    setEditTagIds(prev =>
      prev.includes(tagId) ? prev.filter(id => id !== tagId) : [...prev, tagId]
    );
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
        <Link href="/dashboard" className="inline-block mt-4 text-indigo-600 text-sm">返回列表</Link>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <Link href="/dashboard" className="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 transition">
        <ArrowLeft className="size-4" /> 返回链接列表
      </Link>

      <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
        {/* Header */}
        <div className="p-6 border-b border-gray-100">
          <div className="flex items-center justify-between gap-4">
            {editing ? (
              <input
                type="text" value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                className="flex-1 text-xl font-bold text-gray-900 border-b-2 border-indigo-300 focus:border-indigo-500 focus:outline-none px-2 py-1"
                placeholder="链接标题" autoFocus
              />
            ) : (
              <div className="flex items-center gap-3 flex-1 min-w-0">
                <h1 className="text-xl font-bold text-gray-900 truncate">{link.title || "未命名链接"}</h1>
                <button onClick={handleToggleActive}
                  className={`shrink-0 inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium transition ${
                    link.is_active ? "bg-emerald-50 text-emerald-700" : "bg-gray-100 text-gray-500"
                  }`}
                >
                  {link.is_active ? "启用中" : "已停用"}
                </button>
              </div>
            )}
            {editing ? (
              <div className="flex items-center gap-2 shrink-0">
                <button onClick={() => setEditing(false)} disabled={saving}
                  className="p-2 text-gray-400 hover:text-gray-600 transition"><X className="size-4" /></button>
                <button onClick={handleSave} disabled={saving}
                  className="inline-flex items-center gap-1 px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg transition disabled:opacity-50">
                  <Save className="size-3.5" /> {saving ? "保存中..." : "保存"}
                </button>
              </div>
            ) : (
              <button onClick={() => setEditing(true)}
                className="p-2 text-gray-400 hover:text-indigo-600 transition shrink-0"><Pencil className="size-4" /></button>
            )}
          </div>

          {/* Folder + Tags display */}
          <div className="flex items-center gap-2 mt-2 flex-wrap">
            {link.folder_name && (
              <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-amber-50 text-amber-700">
                <Folder className="size-3" />
                {link.folder_name}
              </span>
            )}
            {(link.tags ?? []).map((t) => (
              <span key={t.id} className="text-xs px-2 py-0.5 rounded-full text-white" style={{ backgroundColor: t.color }}>
                {t.name}
              </span>
            ))}
          </div>
        </div>

        {/* Short URL bar */}
        <div className="px-6 py-4 bg-indigo-50/50 flex items-center gap-2">
          <code className="text-indigo-700 font-mono font-medium text-sm flex-1 truncate">{link.short_url}</code>
          <button onClick={() => copyToClipboard(link.short_url)}
            className={`p-1.5 rounded-lg transition shrink-0 ${copied ? "text-green-500 bg-green-50" : "text-indigo-400 hover:text-indigo-600"}`}>
            {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
          </button>
          <button onClick={() => setShowQR(true)}
            className="p-1.5 rounded-lg text-indigo-400 hover:text-indigo-600 transition shrink-0"><QrCode className="size-3.5" /></button>
          <a href={link.short_url} target="_blank"
            className="p-1.5 rounded-lg text-indigo-400 hover:text-indigo-600 transition shrink-0"><ExternalLink className="size-3.5" /></a>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-3 divide-x border-b border-gray-100">
          <div className="p-4 text-center">
            <p className="text-2xl font-bold text-gray-900 tabular-nums">{link.click_count.toLocaleString()}</p>
            <p className="text-xs text-gray-500 mt-0.5">总点击</p>
          </div>
          <div className="p-4 text-center">
            <p className="text-sm font-bold text-gray-900">{new Date(link.created_at).toLocaleDateString("zh-CN")}</p>
            <p className="text-xs text-gray-500 mt-0.5">创建日期</p>
          </div>
          <div className="p-4 text-center">
            <button
              onClick={() => setShowAnalytics(!showAnalytics)}
              className={`text-sm font-bold transition ${showAnalytics ? "text-indigo-600" : "text-gray-900"}`}
            >
              <BarChart3 className="size-4 inline mr-1" />
              统计
            </button>
            <p className="text-xs text-gray-500 mt-0.5">点击详情</p>
          </div>
        </div>

        {/* Per-link analytics */}
        {showAnalytics && (
          <div className="p-6 border-b border-gray-100 bg-gray-50/50 space-y-4">
            <h3 className="text-sm font-semibold text-gray-900">点击统计</h3>
            {clickData ? (
              <>
                {/* Daily chart */}
                {clickData.daily.length > 0 && (
                  <div>
                    <p className="text-xs text-gray-500 mb-2">每日点击量（近30天）</p>
                    <div className="flex items-end gap-1 h-24">
                      {clickData.daily.map((d) => {
                        const max = Math.max(...clickData.daily.map(dd => dd.count), 1);
                        const h = (d.count / max) * 100;
                        return (
                          <div key={d.date} className="flex-1 flex flex-col items-center gap-1 min-w-0">
                            <div className="w-full flex flex-col justify-end" style={{ height: "80px" }}>
                              <div className="w-full rounded-t bg-indigo-500 min-h-[2px]" style={{ height: `${Math.max(h, 1)}%` }}
                                title={`${d.date}: ${d.count} 点击`} />
                            </div>
                            <span className="text-[9px] text-gray-400 truncate w-full text-center">{d.date.slice(5)}</span>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}
                {/* Platforms */}
                {clickData.platforms.length > 0 && (
                  <div>
                    <p className="text-xs text-gray-500 mb-2">平台分布</p>
                    <div className="space-y-1.5">
                      {clickData.platforms.map((p) => {
                        const total = clickData.platforms.reduce((s, pp) => s + pp.count, 0);
                        const pct = total > 0 ? Math.round((p.count / total) * 100) : 0;
                        return (
                          <div key={p.platform} className="flex items-center gap-2 text-xs">
                            <span className="w-14 text-gray-500">{platformLabels[p.platform] || p.platform}</span>
                            <div className="flex-1 h-1.5 rounded-full bg-gray-200 overflow-hidden">
                              <div className="h-full rounded-full bg-indigo-500" style={{ width: `${pct}%` }} />
                            </div>
                            <span className="w-10 text-right text-gray-500 tabular-nums">{p.count}</span>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}
                {clickData.daily.length === 0 && clickData.platforms.length === 0 && (
                  <p className="text-xs text-gray-400 py-2">暂无点击数据</p>
                )}
              </>
            ) : (
              <div className="flex items-center justify-center py-4">
                <div className="size-5 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
              </div>
            )}
          </div>
        )}

        {/* Edit form */}
        {editing && (
          <div className="p-6 border-b border-gray-100 bg-gray-50/50 space-y-4">
            <div>
              <label className="block text-xs font-medium text-gray-500 mb-1">目标 URL</label>
              <input type="url" value={editOriginalUrl} onChange={(e) => setEditOriginalUrl(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition" />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 mb-1">描述</label>
              <textarea value={editDescription} onChange={(e) => setEditDescription(e.target.value)}
                className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition" rows={2} />
            </div>

            {/* Folder + Tags edit */}
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">文件夹</label>
                <select
                  value={editFolderId ?? ""}
                  onChange={(e) => setEditFolderId(e.target.value ? Number(e.target.value) : null)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition bg-white"
                >
                  <option value="">不分类</option>
                  {folders.map((f: { id: number; name: string }) => (
                    <option key={f.id} value={f.id}>{f.name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">标签</label>
                <div className="flex flex-wrap gap-1">
                  {allTags.map((t: { id: number; name: string; color: string }) => {
                    const selected = editTagIds.includes(t.id);
                    return (
                      <button
                        key={t.id}
                        type="button"
                        onClick={() => toggleEditTag(t.id)}
                        className={`text-xs px-2.5 py-1 rounded-full font-medium transition border ${
                          selected ? "text-white" : "text-gray-500 bg-white border-gray-200 hover:border-gray-300"
                        }`}
                        style={selected ? { backgroundColor: t.color, borderColor: t.color } : {}}
                      >
                        {t.name}
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">密码保护</label>
                <input type="text" value={editPassword} onChange={(e) => setEditPassword(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition"
                  placeholder="留空则不加密" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">过期时间</label>
                <input type="datetime-local" value={editExpiresAt} onChange={(e) => setEditExpiresAt(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition" />
              </div>
            </div>
          </div>
        )}

        {/* Details */}
        <div className="p-6 space-y-2">
          {[
            { label: "短码", value: link.short_code, mono: true },
            { label: "域名", value: link.domain, icon: Globe },
            { label: "描述", value: link.description },
            { label: "创建时间", value: new Date(link.created_at).toLocaleString("zh-CN") },
            { label: "更新时间", value: new Date(link.updated_at).toLocaleString("zh-CN") },
            { label: "过期时间", value: link.expires_at ? new Date(link.expires_at).toLocaleString("zh-CN") : "永不过期", icon: Clock },
            { label: "密码保护", value: link.password_hash ? "已设置" : "未设置", icon: Shield },
          ].filter(({ value }) => value).map(({ label, value, icon: Icon, mono }) => (
            <div key={label} className="flex items-center justify-between py-1.5 text-sm">
              <span className="text-gray-500 flex items-center gap-1.5">
                {Icon && <Icon className="size-3.5" />}{label}
              </span>
              <span className={`text-gray-900 ${mono ? "font-mono" : ""}`}>{value}</span>
            </div>
          ))}

          {/* UTM */}
          {[link.utm_source, link.utm_medium, link.utm_campaign, link.utm_term, link.utm_content].some(Boolean) && (
            <div className="mt-3 pt-3 border-t">
              <p className="text-xs font-medium text-gray-500 mb-2 flex items-center gap-1.5">
                <Tags className="size-3.5" />UTM 参数
              </p>
              {[
                ["来源", link.utm_source], ["媒介", link.utm_medium], ["活动", link.utm_campaign],
                ["关键词", link.utm_term], ["内容", link.utm_content],
              ].filter(([, v]) => v).map(([k, v]) => (
                <div key={k} className="flex items-center justify-between py-1 text-sm">
                  <span className="text-gray-400 text-xs">{k}</span>
                  <code className="text-xs text-gray-700 font-mono">{v}</code>
                </div>
              ))}
            </div>
          )}

          {/* Deep links */}
          {(link.ios_url || link.android_url) && (
            <div className="mt-3 pt-3 border-t">
              <p className="text-xs font-medium text-gray-500 mb-2 flex items-center gap-1.5">
                <Smartphone className="size-3.5" />深度链接
              </p>
              {link.ios_url && <div className="flex justify-between py-1 text-sm"><span className="text-gray-400 text-xs">iOS</span><code className="text-xs text-gray-700">{link.ios_url}</code></div>}
              {link.android_url && <div className="flex justify-between py-1 text-sm"><span className="text-gray-400 text-xs">Android</span><code className="text-xs text-gray-700">{link.android_url}</code></div>}
            </div>
          )}

          {/* Target URL */}
          <div className="mt-3 pt-3 border-t">
            <p className="text-xs font-medium text-gray-500 mb-1">目标 URL</p>
            <a href={link.original_url} target="_blank" className="text-indigo-600 text-sm break-all hover:underline">
              {link.original_url}
            </a>
          </div>
        </div>
      </div>

      {/* QR Code Modal */}
      {showQR && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
          onClick={() => setShowQR(false)}>
          <div className="bg-white rounded-2xl shadow-xl p-6 max-w-sm w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold text-gray-900">二维码</h3>
              <button onClick={() => setShowQR(false)} className="p-1.5 rounded-lg text-gray-400 hover:bg-gray-100 transition"><X className="size-4" /></button>
            </div>
            <div className="flex flex-col items-center gap-4">
              <div className="bg-white border border-gray-100 rounded-xl p-3">
                {qrURL ? <img src={qrURL} alt="QR" className="size-56" />
                  : <div className="size-56 flex items-center justify-center">
                    <div className="size-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" /></div>}
              </div>
              <p className="text-sm font-mono text-gray-600 break-all text-center">{link.short_url}</p>
              {qrURL && (
                <a href={qrURL} download={`kada-${link.short_code}.png`}
                  className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition">
                  <Download className="size-3.5" />下载 PNG
                </a>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
