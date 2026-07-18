"use client";

import { useState, useEffect } from "react";
import { User, Mail, Phone, Pencil, Check, X, Shield, Calendar, Key, Copy, Trash2, Plus } from "lucide-react";
import { toast } from "sonner";
import useSWR from "swr";
import { authAPI, tokensAPI } from "@/lib/api";
import { getToken, getUser, setUser } from "@/lib/auth";

export default function SettingsPage() {
  const token = getToken();
  const savedUser = getUser();

  const [editingName, setEditingName] = useState(false);
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);

  // API Tokens
  const { data: tokenData, mutate: mutateTokens } = useSWR(
    token ? "api-tokens" : null,
    () => tokensAPI.list(token!)
  );
  const tokens = tokenData?.tokens ?? [];
  const [newTokenName, setNewTokenName] = useState("");
  const [creating, setCreating] = useState(false);
  const [newToken, setNewToken] = useState<string | null>(null);

  useEffect(() => {
    if (savedUser?.name) {
      setName(savedUser.name);
    }
  }, [savedUser?.name]);

  const handleSaveName = async () => {
    if (!token || !name.trim()) return;
    setSaving(true);
    try {
      const data = await authAPI.updateMe(token, { name: name.trim() });
      if (data.user) {
        setUser(data.user);
      }
      setEditingName(false);
      toast.success("姓名已更新");
    } catch {
      toast.error("更新失败");
    } finally {
      setSaving(false);
    }
  };

  const handleCreateToken = async () => {
    if (!token || !newTokenName.trim()) return;
    setCreating(true);
    try {
      const data = await tokensAPI.create(token, newTokenName.trim());
      setNewToken(data.token);
      setNewTokenName("");
      mutateTokens();
      toast.success("Token 创建成功");
    } catch {
      toast.error("创建失败");
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteToken = async (id: number) => {
    if (!token) return;
    try {
      await tokensAPI.delete(token, id);
      mutateTokens();
      toast.success("Token 已删除");
    } catch {
      toast.error("删除失败");
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">设置</h1>
        <p className="text-sm text-gray-500 mt-1">管理你的个人信息</p>
      </div>

      {/* Profile Card */}
      <div className="rounded-xl border border-gray-100 bg-white shadow-sm overflow-hidden">
        <div className="border-b border-gray-100 px-6 py-4">
          <div className="flex items-center gap-2">
            <User className="size-4 text-gray-500" />
            <h2 className="font-semibold text-gray-900">个人信息</h2>
          </div>
        </div>

        <div className="px-6 py-5 space-y-5">
          <div className="flex items-center gap-4">
            <div className="flex size-14 items-center justify-center rounded-full bg-indigo-100 text-xl font-bold text-indigo-600">
              {(savedUser?.name || savedUser?.email || "U")[0].toUpperCase()}
            </div>
            <div>
              <p className="text-sm font-medium text-gray-900">
                {savedUser?.name || savedUser?.email || "用户"}
              </p>
              <p className="text-xs text-gray-500">ID: {savedUser?.id}</p>
            </div>
          </div>

          {/* Name */}
          <div className="flex items-center justify-between py-2">
            <div className="flex items-center gap-3 min-w-0">
              <User className="size-4 text-gray-400 shrink-0" />
              <div className="min-w-0">
                <p className="text-xs text-gray-500">姓名</p>
                {editingName ? (
                  <div className="flex items-center gap-2 mt-1">
                    <input
                      type="text" value={name}
                      onChange={(e) => setName(e.target.value)}
                      className="border border-gray-200 rounded-lg px-2 py-1 text-sm text-gray-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent w-48"
                      autoFocus
                      onKeyDown={(e) => {
                        if (e.key === "Enter") handleSaveName();
                        if (e.key === "Escape") { setEditingName(false); setName(savedUser?.name || ""); }
                      }}
                    />
                    <button onClick={handleSaveName} disabled={saving} className="p-1 rounded text-emerald-600 hover:bg-emerald-50 transition">
                      <Check className="size-3.5" />
                    </button>
                    <button onClick={() => { setEditingName(false); setName(savedUser?.name || ""); }}
                      className="p-1 rounded text-gray-400 hover:bg-gray-100 transition">
                      <X className="size-3.5" />
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2 mt-1">
                    <p className="text-sm text-gray-900">
                      {savedUser?.name || <span className="text-gray-400 italic">未设置</span>}
                    </p>
                    <button onClick={() => { setEditingName(true); setName(savedUser?.name || ""); }}
                      className="p-1 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition">
                      <Pencil className="size-3" />
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>

          <div className="flex items-center gap-3 py-2">
            <Mail className="size-4 text-gray-400 shrink-0" />
            <div><p className="text-xs text-gray-500">邮箱</p><p className="text-sm text-gray-900">{savedUser?.email || <span className="text-gray-400 italic">未绑定</span>}</p></div>
          </div>
          <div className="flex items-center gap-3 py-2">
            <Phone className="size-4 text-gray-400 shrink-0" />
            <div><p className="text-xs text-gray-500">手机号</p><p className="text-sm text-gray-900">{savedUser?.phone || <span className="text-gray-400 italic">未绑定</span>}</p></div>
          </div>
        </div>
      </div>

      {/* API Tokens Card */}
      <div className="rounded-xl border border-gray-100 bg-white shadow-sm overflow-hidden">
        <div className="border-b border-gray-100 px-6 py-4">
          <div className="flex items-center gap-2">
            <Key className="size-4 text-gray-500" />
            <h2 className="font-semibold text-gray-900">API Tokens</h2>
          </div>
        </div>

        <div className="px-6 py-5 space-y-4">
          {/* New token shown once */}
          {newToken && (
            <div className="rounded-xl bg-amber-50 border border-amber-200 p-4">
              <p className="text-sm font-medium text-amber-800 mb-2">新 Token 已创建，仅显示一次，请立即复制保存：</p>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-white rounded-lg px-3 py-2 text-sm font-mono text-gray-800 border border-amber-200 break-all">
                  {newToken}
                </code>
                <button
                  onClick={() => { navigator.clipboard.writeText(newToken); toast.success("已复制"); }}
                  className="p-2 rounded-lg text-amber-600 hover:bg-amber-100 transition shrink-0"
                >
                  <Copy className="size-4" />
                </button>
              </div>
              <button onClick={() => setNewToken(null)}
                className="mt-2 text-xs text-amber-600 hover:text-amber-700">
                我已保存，关闭提示
              </button>
            </div>
          )}

          {/* Create new */}
          <div className="flex items-center gap-2">
            <input
              type="text" value={newTokenName}
              onChange={(e) => setNewTokenName(e.target.value)}
              className="flex-1 rounded-lg border border-gray-200 px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition"
              placeholder="Token 名称，如：生产环境、本地开发"
              onKeyDown={(e) => { if (e.key === "Enter") handleCreateToken(); }}
            />
            <button
              onClick={handleCreateToken}
              disabled={creating || !newTokenName.trim()}
              className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed transition shrink-0"
            >
              <Plus className="size-3.5" /> 创建
            </button>
          </div>

          {/* Token list */}
          {tokens.length > 0 ? (
            <div className="space-y-1">
              {tokens.map((t: { id: number; name: string; last_used?: string; created_at: string }) => (
                <div key={t.id} className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-gray-50 transition">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{t.name}</p>
                    <p className="text-xs text-gray-400">
                      创建于 {new Date(t.created_at).toLocaleDateString("zh-CN")}
                      {t.last_used ? ` · 最近使用 ${new Date(t.last_used).toLocaleDateString("zh-CN")}` : " · 从未使用"}
                    </p>
                  </div>
                  <button
                    onClick={() => handleDeleteToken(t.id)}
                    className="p-1.5 rounded-lg text-gray-400 hover:text-red-500 hover:bg-red-50 transition"
                  >
                    <Trash2 className="size-3.5" />
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-400 py-2">暂无 API Token，创建一个用于外部程序调用 API</p>
          )}
        </div>
      </div>

      {/* Account Info Card */}
      <div className="rounded-xl border border-gray-100 bg-white shadow-sm overflow-hidden">
        <div className="border-b border-gray-100 px-6 py-4">
          <div className="flex items-center gap-2">
            <Shield className="size-4 text-gray-500" />
            <h2 className="font-semibold text-gray-900">账号信息</h2>
          </div>
        </div>
        <div className="px-6 py-5 space-y-4">
          <div className="flex items-center gap-3 py-2">
            <Calendar className="size-4 text-gray-400 shrink-0" />
            <div><p className="text-xs text-gray-500">注册时间</p><p className="text-sm text-gray-900">{savedUser?.created_at ? new Date(savedUser.created_at).toLocaleDateString("zh-CN", { year: "numeric", month: "long", day: "numeric" }) : "—"}</p></div>
          </div>
          <div className="flex items-center gap-3 py-2">
            <Shield className="size-4 text-gray-400 shrink-0" />
            <div><p className="text-xs text-gray-500">登录方式</p><p className="text-sm text-gray-900">{savedUser?.phone ? "手机号验证码" : "邮箱密码"}</p></div>
          </div>
        </div>
      </div>
    </div>
  );
}
