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
  const [tab, setTab] = useState<"phone" | "email">("phone");
  const [loading, setLoading] = useState(false);

  // 手机号注册
  const [phoneName, setPhoneName] = useState("");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [sending, setSending] = useState(false);
  const [phoneError, setPhoneError] = useState("");
  const [codeError, setCodeError] = useState("");

  // 邮箱注册
  const [emailName, setEmailName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [emailErrors, setEmailErrors] = useState<{
    name?: string;
    email?: string;
    password?: string;
    confirm?: string;
  }>({});

  const phoneValid = phone.length === 11;

  const handleSendCode = async () => {
    if (phone.length !== 11) {
      setPhoneError("请输入 11 位手机号");
      return;
    }
    setPhoneError("");
    setSending(true);
    try {
      await authAPI.sendSMSCode(phone);
      setCodeSent(true);
      toast.success("验证码已发送");
      setCountdown(60);
      const timer = setInterval(() => {
        setCountdown((c) => {
          if (c <= 1) {
            clearInterval(timer);
            return 0;
          }
          return c - 1;
        });
      }, 1000);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "发送失败");
    } finally {
      setSending(false);
    }
  };

  const handlePhoneRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    let ok = true;
    if (phone.length !== 11) {
      setPhoneError("请输入 11 位手机号");
      ok = false;
    } else {
      setPhoneError("");
    }
    if (code.length !== 6) {
      setCodeError("请输入 6 位验证码");
      ok = false;
    } else {
      setCodeError("");
    }
    if (!ok) return;

    setLoading(true);
    try {
      const data = await authAPI.loginByPhone(phone, code);
      // 昵称：输入了就用输入的，否则默认「用户+手机号后4位」
      const nickname = phoneName.trim() || `用户${phone.slice(-4)}`;
      if (!data.user.name) {
        await authAPI.updateMe(data.token, { name: nickname });
      }
      setToken(data.token);
      setUser({ ...data.user, name: data.user.name || nickname });
      toast.success("注册成功！");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "注册失败");
    } finally {
      setLoading(false);
    }
  };

  const handleEmailRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    const errs: typeof emailErrors = {};
    if (!emailName.trim()) errs.name = "请输入昵称";
    if (!EMAIL_RE.test(email)) errs.email = "请输入有效的邮箱地址";
    if (password.length < 6) errs.password = "密码至少 6 位";
    if (confirm !== password) errs.confirm = "两次输入的密码不一致";
    setEmailErrors(errs);
    if (Object.keys(errs).length > 0) return;

    setLoading(true);
    try {
      const data = await authAPI.registerByEmail(email, password, emailName.trim());
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
          <Link href="/login" className="font-medium text-indigo-300 hover:text-indigo-200">
            立即登录
          </Link>
        </>
      }
    >
      {/* 手机号 / 邮箱 分段切换 */}
      <div className="mb-6 grid grid-cols-2 gap-1 rounded-lg bg-white/10 p-1">
        <button
          type="button"
          onClick={() => setTab("phone")}
          className={`rounded-md py-2 text-sm font-medium transition ${
            tab === "phone"
              ? "bg-white text-neutral-900 shadow-sm"
              : "text-neutral-300 hover:text-white"
          }`}
        >
          手机号注册
        </button>
        <button
          type="button"
          onClick={() => setTab("email")}
          className={`rounded-md py-2 text-sm font-medium transition ${
            tab === "email"
              ? "bg-white text-neutral-900 shadow-sm"
              : "text-neutral-300 hover:text-white"
          }`}
        >
          邮箱注册
        </button>
      </div>

      {tab === "phone" ? (
        <form onSubmit={handlePhoneRegister} className="space-y-4" noValidate>
          <FormField id="pname" label="昵称">
            <input
              id="pname"
              type="text"
              value={phoneName}
              onChange={(e) => setPhoneName(e.target.value)}
              className={`${inputBase} ${fieldState(false)}`}
              placeholder="你的昵称（可选）"
            />
          </FormField>
          <FormField id="pphone" label="手机号" error={phoneError}>
            <input
              id="pphone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value.replace(/\D/g, "").slice(0, 11))}
              className={`${inputBase} ${fieldState(!!phoneError)}`}
              placeholder="请输入 11 位手机号"
              required
            />
          </FormField>
          <FormField id="pcode" label="验证码" error={codeError}>
            <div className="flex gap-3">
              <input
                id="pcode"
                type="text"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                className={`${inputBase} ${fieldState(!!codeError)}`}
                placeholder="6 位验证码"
                required
              />
              <button
                type="button"
                onClick={handleSendCode}
                disabled={countdown > 0 || !phoneValid || sending}
                className="shrink-0 rounded-lg bg-indigo-500/20 px-4 py-2.5 text-sm font-medium text-indigo-200 transition hover:bg-indigo-500/30 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {countdown > 0 ? `${countdown}s` : codeSent ? "重新发送" : "获取验证码"}
              </button>
            </div>
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "注册中..." : "注册"}
          </button>
        </form>
      ) : (
        <form onSubmit={handleEmailRegister} className="space-y-4" noValidate>
          <FormField id="name" label="昵称" error={emailErrors.name}>
            <input
              id="name"
              type="text"
              value={emailName}
              onChange={(e) => setEmailName(e.target.value)}
              className={`${inputBase} ${fieldState(!!emailErrors.name)}`}
              placeholder="你的昵称"
              required
            />
          </FormField>
          <FormField id="email" label="邮箱" error={emailErrors.email}>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={`${inputBase} ${fieldState(!!emailErrors.email)}`}
              placeholder="your@email.com"
              required
            />
          </FormField>
          <FormField id="password" label="密码" error={emailErrors.password}>
            <PasswordInput
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={fieldState(!!emailErrors.password)}
              placeholder="至少 6 位"
              required
              minLength={6}
            />
          </FormField>
          <FormField id="confirm" label="确认密码" error={emailErrors.confirm}>
            <PasswordInput
              id="confirm"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              className={fieldState(!!emailErrors.confirm)}
              placeholder="再次输入密码"
              required
            />
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "注册中..." : "注册"}
          </button>
        </form>
      )}
    </AuthCard>
  );
}
