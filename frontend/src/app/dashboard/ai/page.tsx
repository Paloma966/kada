"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { MessageSquare, Plus, Send, Sparkles, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { aiAPI } from "@/lib/api";
import { useT } from "@/lib/i18n";
import { cn } from "@/lib/utils";

interface Msg {
  role: "user" | "assistant";
  content: string;
}

interface Conv {
  conversation_id: string;
  created_at: string;
}

// 时间显示：2026-09-16 14:30（本地时区）
function fmtTime(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export default function AIPage() {
  const t = useT();
  const [messages, setMessages] = useState<Msg[]>([]);   // 当前聊天的消息
  const [input, setInput] = useState("");                 // 输入框
  const [conversationId, setConversationId] = useState<string | null>(null); // 当前会话 id
  const [loading, setLoading] = useState(false);          // 等 AI 回复中

  // 历史会话
  const [convs, setConvs] = useState<Conv[]>([]);
  const [listLoading, setListLoading] = useState(false);
  const [activeId, setActiveId] = useState<string | null>(null); // 当前选中的会话

  const bottomRef = useRef<HTMLDivElement>(null);

  // 从后端拉历史会话列表（进页面时 + 每次对话后刷新）
  const refreshList = useCallback(async () => {
    setListLoading(true);
    try {
      const data = await aiAPI.listConversations();
      setConvs(data.conversations ?? []);
    } catch {
      toast.error(t("历史会话加载失败"));
    } finally {
      setListLoading(false);
    }
  }, [t]);

  useEffect(() => {
    refreshList();
  }, [refreshList]);

  // 新消息自动滚到底部
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // 新对话：清空本地状态（下次发送时后端会自动建新会话）
  const newChat = () => {
    setMessages([]);
    setConversationId(null);
    setActiveId(null);
    setInput("");
  };

  // 点击历史会话：拉它的消息，接着聊
  const openConversation = async (id: string) => {
    try {
      const data = await aiAPI.getMessages(id);
      const msgs: Msg[] = (data.messages ?? []).map(
        (m: { role: string; content: string }) => ({
          role: m.role === "user" ? "user" : "assistant",
          content: m.content,
        })
      );
      setMessages(msgs);
      setConversationId(id);
      setActiveId(id);
    } catch {
      toast.error(t("加载消息失败"));
    }
  };

  // 删除会话：后端级联删消息，前端刷新列表
  const deleteConversation = async (id: string) => {
    try {
      await aiAPI.deleteConversation(id);
      setConvs((prev) => prev.filter((c) => c.conversation_id !== id));
      if (id === activeId) {
        // 删的是当前会话 → 回到空的新对话状态
        setMessages([]);
        setConversationId(null);
        setActiveId(null);
      }
      toast.success(t("已删除"));
    } catch {
      toast.error(t("删除失败"));
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
      // 会话结束：记住会话 id（下一轮续聊），刷新历史列表
      const cid = (data.conversation_id as string) ?? null;
      setConversationId(cid);
      setActiveId(cid);
      refreshList();
    } else if (event === "error") {
      toast.error((data.message as string) || t("出错了"));
    }
  }

  return (
    <div className="flex h-full">
      {/* ===== 左侧：历史会话列表（贴着左边菜单栏） ===== */}
      <aside className="flex w-56 shrink-0 flex-col border-r border-gray-100 bg-white">
        <div className="flex h-12 shrink-0 items-center justify-between border-b border-gray-100 px-3">
          <span className="text-sm font-semibold text-gray-700">{t("历史会话")}</span>
          <button
            onClick={newChat}
            className="rounded-lg p-1 text-gray-400 hover:bg-gray-50 hover:text-indigo-600 transition"
            title={t("新对话")}
          >
            <Plus className="size-4" />
          </button>
        </div>

        <div className="flex-1 space-y-1 overflow-y-auto p-2">
          {listLoading && convs.length === 0 ? (
            <p className="px-2 py-3 text-xs text-gray-400">{t("加载中...")}</p>
          ) : convs.length === 0 ? (
            <p className="px-2 py-3 text-xs text-gray-400">{t("还没有历史会话")}</p>
          ) : (
            convs.map((c) => (
              <div
                key={c.conversation_id}
                onClick={() => openConversation(c.conversation_id)}
                className={cn(
                  "group flex cursor-pointer items-center justify-between rounded-lg px-2 py-2 text-xs transition",
                  c.conversation_id === activeId
                    ? "bg-indigo-50 text-indigo-700"
                    : "text-gray-600 hover:bg-gray-50"
                )}
              >
                <span className="flex min-w-0 items-center gap-1.5">
                  <MessageSquare className="size-3.5 shrink-0 opacity-60" />
                  <span className="truncate">{fmtTime(c.created_at)}</span>
                </span>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    deleteConversation(c.conversation_id);
                  }}
                  className="shrink-0 text-gray-400 opacity-0 transition hover:text-red-500 group-hover:opacity-100"
                  title={t("删除会话")}
                >
                  <Trash2 className="size-3.5" />
                </button>
              </div>
            ))
          )}
        </div>
      </aside>

      {/* ===== 右侧：聊天区（占满剩余全部空间） ===== */}
      <div className="flex min-w-0 flex-1 flex-col px-4 sm:px-6">
        {/* 头部：标题 + 新对话按钮 */}
        <div className="flex items-center justify-between pt-2">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{t("AI 助手")}</h1>
            <p className="mt-1 text-sm text-gray-500">{t("问任何关于你链接和数据的问题")}</p>
          </div>
          <button
            onClick={newChat}
            disabled={loading}
            className="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-40 transition"
          >
            <Plus className="size-3.5" />
            {t("新对话")}
          </button>
        </div>

        {/* 消息区 */}
        <div className="mt-4 flex-1 overflow-y-auto rounded-xl border border-gray-100 bg-white p-4 shadow-sm">
          {messages.length === 0 ? (
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
        <div className="mt-3 flex items-end gap-2 pb-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSend();
              }
            }}
            rows={2}
            placeholder={t("输入消息...（Enter 发送）")}
            className="flex-1 resize-none rounded-xl border border-gray-200 p-3 text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
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
    </div>
  );
}
