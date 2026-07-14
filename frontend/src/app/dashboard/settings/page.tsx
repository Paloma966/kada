"use client";

import { useState, useEffect } from "react";
import { User, Mail, Phone, Pencil, Check, X, Shield, Calendar } from "lucide-react";
import { toast } from "sonner";
import { authAPI } from "@/lib/api";
import { getToken, getUser, setUser } from "@/lib/auth";

export default function SettingsPage() {
  const token = getToken();
  const savedUser = getUser();

  const [editingName, setEditingName] = useState(false);
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);

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
      // Update stored user
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
          {/* Avatar */}
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
                      type="text"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      className="border border-gray-200 rounded-lg px-2 py-1 text-sm text-gray-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent w-48"
                      autoFocus
                      onKeyDown={(e) => {
                        if (e.key === "Enter") handleSaveName();
                        if (e.key === "Escape") {
                          setEditingName(false);
                          setName(savedUser?.name || "");
                        }
                      }}
                    />
                    <button
                      onClick={handleSaveName}
                      disabled={saving}
                      className="p-1 rounded text-emerald-600 hover:bg-emerald-50 transition"
                    >
                      <Check className="size-3.5" />
                    </button>
                    <button
                      onClick={() => {
                        setEditingName(false);
                        setName(savedUser?.name || "");
                      }}
                      className="p-1 rounded text-gray-400 hover:bg-gray-100 transition"
                    >
                      <X className="size-3.5" />
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2 mt-1">
                    <p className="text-sm text-gray-900">
                      {savedUser?.name || <span className="text-gray-400 italic">未设置</span>}
                    </p>
                    <button
                      onClick={() => {
                        setEditingName(true);
                        setName(savedUser?.name || "");
                      }}
                      className="p-1 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition"
                    >
                      <Pencil className="size-3" />
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Email */}
          <div className="flex items-center gap-3 py-2">
            <Mail className="size-4 text-gray-400 shrink-0" />
            <div>
              <p className="text-xs text-gray-500">邮箱</p>
              <p className="text-sm text-gray-900">
                {savedUser?.email || <span className="text-gray-400 italic">未绑定</span>}
              </p>
            </div>
          </div>

          {/* Phone */}
          <div className="flex items-center gap-3 py-2">
            <Phone className="size-4 text-gray-400 shrink-0" />
            <div>
              <p className="text-xs text-gray-500">手机号</p>
              <p className="text-sm text-gray-900">
                {savedUser?.phone || <span className="text-gray-400 italic">未绑定</span>}
              </p>
            </div>
          </div>
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
            <div>
              <p className="text-xs text-gray-500">注册时间</p>
              <p className="text-sm text-gray-900">
                {savedUser?.created_at
                  ? new Date(savedUser.created_at).toLocaleDateString("zh-CN", {
                      year: "numeric",
                      month: "long",
                      day: "numeric",
                    })
                  : "—"}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3 py-2">
            <Shield className="size-4 text-gray-400 shrink-0" />
            <div>
              <p className="text-xs text-gray-500">登录方式</p>
              <p className="text-sm text-gray-900">
                {savedUser?.phone ? "手机号验证码" : "邮箱密码"}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
