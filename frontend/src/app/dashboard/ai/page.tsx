"use client";

import { useEffect, useRef, useState } from "react";
import { RotateCcw, Send, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { aiAPI } from "@/lib/api";
import { useT } from "@/lib/i18n";
import { cn } from "@/lib/utils";

interface Msg {
  role: "user" | "assistant";
  content: string;
}

export default function AIPage() {
  const t = useT();
  const [messages, setMessages] = useState<Msg[]>([]);   // 当前聊天的消息
  const [input, setInput] = useState("");                 // 输入框
  const [conversationId, setConversationId] = useState<string | null>(null); // 当前会话 id
  const [loading, setLoading] = useState(false);          // 等 AI 回复中
  const [initLoading, setInitLoading] = useState(true);   // 进页面加载上次会话中

  const bottomRef = useRef<HTMLDivElement>(null);

  // 进页面：加载当前会话（最新一个），自动续上上次的聊天；没有就是空白新会话
  useEffect(() => {
    (async () => {
      try {
        const data = await aiAPI.getCurrentSession();
        if (data.conversation_id) {
          setConversationId(data.conversation_id);
          setMessages(
            (data.messages ?? []).map(
              (m: { role: string; content: string }): Msg => ({
                role: m.role === "user" ? "user" : "assistant",
                content: m.content,
              })
            )
          );
        }
      } catch {
        // 加载失败就当新会话，不打扰用户
      } finally {
        setInitLoading(false);
      }
    })();
  }, []);

  // 新消息自动滚到底部
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // 重新开始：后端删掉旧会话、建新空会话；前端清空界面
  const restart = async () => {
    if (loading || initLoading) return;
    try {
      const data = await aiAPI.restartSession();
      setMessages([]);
      setConversationId(data.conversation_id ?? null);
      setInput("");
      toast.success(t("已开始新对话"));
    } catch {
      toast.error(t("操作失败，请重试"));
    }
  };

  const handleSend = async () => {
    const text = input.trim();
    if (!text || loading) return;
    setInput("");
    setLoading(true);

    // 1. 先显示用户这句
    setMessages((prev) => [...prev, { role: "user", content: text }]);
    // 2. 加一个空的 AI 气泡，后面逐字填
    setMessages((prev) => [...prev, { role: "assistant", content: "" }]);

    try {
      const res = await aiAPI.streamChat({
        conversation_id: conversationId ?? undefined,
        message: text,
      });

      if (!res.ok) {
        const body = await res.text();
        throw new Error(body || `HTTP ${res.status}`);
      }

      // 3. 逐块读 SSE 流
      const reader = res.body!.getReader();
      const decoder = new TextDecoder();
      let buf = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += decoder.decode(value, { stream: true });

        // SSE 消息以空行结尾，按 "\n\n" 切块
        let idx: number;
        while ((idx = buf.indexOf("\n\n")) !== -1) {
          const block = buf.slice(0, idx);
          buf = buf.slice(idx + 2);
          handleBlock(block);
        }
      }
    } catch (e) {
      setMessages((prev) => prev.slice(0, -1));
      toast.error(e instanceof Error ? e.message : t("出错了"));
    } finally {
      setLoading(false);
    }
  };

  // 处理一条完整 SSE 消息：event: xxx\ndata: {json}
  function handleBlock(block: string) {
    const lines = block.split("\n");
    const event = lines.find((l) => l.startsWith("event:"))?.slice(6).trim();
    const dataLine = lines.find((l) => l.startsWith("data:"))?.slice(5).trim();
    if (!dataLine) return;

    let data: Record<string, unknown>;
    try {
      data = JSON.parse(dataLine);
    } catch {
      return;
    }

    if (event === "token" && typeof data.delta === "string") {
      // 打字效果：新字追加到最后一个 AI 气泡
      setMessages((prev) => {
        const next = [...prev];
        const last = next[next.length - 1];
        next[next.length - 1] = { role: "assistant", content: last.content + data.delta };
        return next;
      });
    } else if (event === "done") {
      // 会话结束：记住会话 id（下一轮续聊）
      const cid = (data.conversation_id as string) ?? null;
      setConversationId(cid);
    } else if (event === "error") {
      toast.error((data.message as string) || t("出错了"));
    }
  }

  return (
    <div className="flex h-[70vh] flex-col">
      {/* 头部：标题 + 重新开始 */}
      <div className="flex shrink-0 items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{t("AI 助手")}</h1>
          <p className="mt-1 text-sm text-gray-500">{t("问任何关于你链接和数据的问题")}</p>
        </div>
        <button
          onClick={restart}
          disabled={loading || initLoading}
          className="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-40 transition"
        >
          <RotateCcw className="size-3.5" />
          {t("重新开始")}
        </button>
      </div>

      {/* 消息区 */}
      <div className="mt-4 flex-1 overflow-y-auto rounded-xl border border-gray-100 bg-white p-4 shadow-sm">
        {initLoading ? (
          <div className="flex h-full items-center justify-center text-sm text-gray-400">
            {t("加载中...")}
          </div>
        ) : messages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center text-center text-gray-400">
            <div className="mb-3 flex size-14 items-center justify-center rounded-2xl bg-indigo-50">
              <Sparkles className="size-7 text-indigo-500" />
            </div>
            <h3 className="text-lg font-semibold text-gray-700">{t("开始对话")}</h3>
            <p className="mt-1 text-sm">{t("AI 会记住本次对话的上下文")}</p>
          </div>
        ) : (
          <div className="space-y-3">
            {messages.map((m, i) => (
              <div
                key={i}
                className={cn("flex", m.role === "user" ? "justify-end" : "justify-start")}
              >
                <div
                  className={cn(
                    "max-w-[80%] whitespace-pre-wrap break-words rounded-2xl px-3 py-2 text-sm leading-relaxed",
                    m.role === "user"
                      ? "bg-indigo-600 text-white"
                      : "bg-gray-100 text-gray-900"
                  )}
                >
                  {m.content || (loading && i === messages.length - 1 ? "…" : "")}
                </div>
              </div>
            ))}
            <div ref={bottomRef} />
          </div>
        )}
      </div>

      {/* 输入区 */}
      <div className="mt-3 flex shrink-0 items-end gap-2">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              handleSend();
            }
          }}
          rows={1}
          placeholder={t("输入消息...（Enter 发送）")}
          className="h-11 flex-1 resize-none rounded-xl border border-gray-200 px-3 py-2.5 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
        <button
          onClick={handleSend}
          disabled={!input.trim() || loading}
          className="inline-flex size-11 shrink-0 items-center justify-center rounded-xl bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed transition"
        >
          <Send className="size-4" />
        </button>
      </div>
    </div>
  );
}
