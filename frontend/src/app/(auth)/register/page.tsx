"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { authAPI } from "@/lib/api";
import { setToken, setUser } from "@/lib/auth";
import { AuthCard } from "@/components/auth/AuthCard";
import { FormField, inputBase, fieldState } from "@/components/auth/FormField";
import { PasswordInput } from "@/components/auth/PasswordInput";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function RegisterPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [errors, setErrors] = useState<{
    name?: string;
    email?: string;
    password?: string;
    confirm?: string;
  }>({});

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const errs: typeof errors = {};
    if (!name.trim()) errs.name = "请输入昵称";
    if (!EMAIL_RE.test(email)) errs.email = "请输入有效的邮箱地址";
    if (password.length < 6) errs.password = "密码至少 6 位";
    if (confirm !== password) errs.confirm = "两次输入的密码不一致";
    setErrors(errs);
    if (Object.keys(errs).length > 0) return;

    setLoading(true);
    try {
      const data = await authAPI.registerByEmail(email, password, name.trim());
      setToken(data.token);
      setUser(data.user);
      toast.success("注册成功！");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "注册失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthCard
      title="注册 Kada"
      subtitle="创建你的短链接管理账号"
      footer={
        <>
          已有账号？{" "}
          <Link href="/login" className="font-medium text-indigo-600 hover:text-indigo-500">
            立即登录
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <FormField id="name" label="昵称" error={errors.name}>
          <input
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={`${inputBase} ${fieldState(!!errors.name)}`}
            placeholder="你的昵称"
            required
          />
        </FormField>
        <FormField id="email" label="邮箱" error={errors.email}>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className={`${inputBase} ${fieldState(!!errors.email)}`}
            placeholder="your@email.com"
            required
          />
        </FormField>
        <FormField id="password" label="密码" error={errors.password}>
          <PasswordInput
            id="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className={fieldState(!!errors.password)}
            placeholder="至少 6 位"
            required
            minLength={6}
          />
        </FormField>
        <FormField id="confirm" label="确认密码" error={errors.confirm}>
          <PasswordInput
            id="confirm"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            className={fieldState(!!errors.confirm)}
            placeholder="再次输入密码"
            required
          />
        </FormField>
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {loading ? "注册中..." : "注册"}
        </button>
      </form>
    </AuthCard>
  );
}
