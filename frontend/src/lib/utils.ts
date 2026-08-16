export function cn(...classes: (string | boolean | undefined | null)[]): string {
  return classes.filter(Boolean).join(" ");
}

/**
 * safeHref 只允许 http/https 链接用于 <a href>。
 * 后端已拒绝 javascript:/data: 协议，这里再兜底存量数据，
 * 防止点击时在面板源内执行脚本；非法值返回 "#"。
 */
export function safeHref(url: string | null | undefined): string {
  if (!url) return "#";
  try {
    const u = new URL(url);
    if (u.protocol === "http:" || u.protocol === "https:") {
      return url;
    }
  } catch {
    // 无法解析按不安全处理
  }
  return "#";
}
