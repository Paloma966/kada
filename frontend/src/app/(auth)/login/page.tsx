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
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import { useT } from "@/lib/i18n";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function LoginPage() {
  const router = useRouter();
  const t = useT();
  const [tab, setTab] = useState<"phone" | "email">("phone");
  const [loading, setLoading] = useState(false);

  // Phone number sign-in
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [sending, setSending] = useState(false);
  const [phoneError, setPhoneError] = useState("");
  const [codeError, setCodeError] = useState("");

  // Email sign-in
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState("");
  const [passwordError, setPasswordError] = useState("");

  const phoneValid = phone.length === 11;

  const handleSendCode = async () => {
    if (phone.length !== 11) {
      setPhoneError(t("请输入 11 位手机号"));
      return;
    }
    setPhoneError("");
    setSending(true);
    try {
      await authAPI.sendSMSCode(phone);
      setCodeSent(true);
      toast.success(t("验证码已发送"));
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
      toast.error(e instanceof Error ? e.message : t("发送失败"));
    } finally {
      setSending(false);
    }
  };

  const handlePhoneLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    let ok = true;
    if (phone.length !== 11) {
      setPhoneError(t("请输入 11 位手机号"));
      ok = false;
    } else {
      setPhoneError("");
    }
    if (code.length !== 6) {
      setCodeError(t("请输入 6 位验证码"));
      ok = false;
    } else {
      setCodeError("");
    }
    if (!ok) return;

    setLoading(true);
    try {
      const data = await authAPI.loginByPhone(phone, code);
      setToken(data.token);
      setUser(data.user);
      toast.success(t("登录成功"));
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t("登录失败"));
    } finally {
      setLoading(false);
    }
  };

  const handleEmailLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    let ok = true;
    if (!EMAIL_RE.test(email)) {
      setEmailError(t("请输入有效的邮箱地址"));
      ok = false;
    } else {
      setEmailError("");
    }
    if (!password) {
      setPasswordError(t("请输入密码"));
      ok = false;
    } else {
      setPasswordError("");
    }
    if (!ok) return;

    setLoading(true);
    try {
      const data = await authAPI.loginByEmail(email, password);
      setToken(data.token);
      setUser(data.user);
      toast.success(t("登录成功"));
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t("登录失败"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <div className="fixed right-4 top-4 z-50">
        <LanguageSwitcher />
      </div>
      <AuthCard
        title={t("登录 Kada")}
        subtitle={t("智能短链接管理平台")}
        footer={
          <>
            {t("还没有账号？")}{" "}
            <Link href="/register" className="font-medium text-indigo-300 hover:text-indigo-200">
              {t("立即注册")}
            </Link>
          </>
        }
      >
        {/* Phone / email segmented switch */}
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
            {t("手机号登录")}
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
            {t("邮箱登录")}
          </button>
        </div>

        {tab === "phone" ? (
          <form onSubmit={handlePhoneLogin} className="space-y-4" noValidate>
            <FormField id="phone" label={t("手机号")} error={phoneError}>
              <input
                id="phone"
                type="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value.replace(/\D/g, "").slice(0, 11))}
                className={`${inputBase} ${fieldState(!!phoneError)}`}
                placeholder={t("请输入 11 位手机号")}
                required
              />
            </FormField>
            <FormField id="code" label={t("验证码")} error={codeError}>
              <div className="flex gap-3">
                <input
                  id="code"
                  type="text"
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                  className={`${inputBase} ${fieldState(!!codeError)}`}
                  placeholder={t("6 位验证码")}
                  required
                />
                <button
                  type="button"
                  onClick={handleSendCode}
                  disabled={countdown > 0 || !phoneValid || sending}
                  className="shrink-0 rounded-lg bg-indigo-500/20 px-4 py-2.5 text-sm font-medium text-indigo-200 transition hover:bg-indigo-500/30 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {countdown > 0 ? `${countdown}s` : codeSent ? t("重新发送") : t("获取验证码")}
                </button>
              </div>
            </FormField>
            <button
              type="submit"
              disabled={loading}
              className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? t("登录中...") : t("登录")}
            </button>
          </form>
        ) : (
          <form onSubmit={handleEmailLogin} className="space-y-4" noValidate>
            <FormField id="email" label={t("邮箱")} error={emailError}>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className={`${inputBase} ${fieldState(!!emailError)}`}
                placeholder="your@email.com"
                required
              />
            </FormField>
            <FormField id="password" label={t("密码")} error={passwordError}>
              <PasswordInput
                id="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={fieldState(!!passwordError)}
                placeholder={t("请输入密码")}
                required
              />
            </FormField>
            <button
              type="submit"
              disabled={loading}
              className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? t("登录中...") : t("登录")}
            </button>
          </form>
        )}
      </AuthCard>
    </>
  );
}
