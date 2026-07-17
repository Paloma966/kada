"use client";

import { Copy, ExternalLink, Pencil, Trash2, BarChart3, Check, QrCode, Download, X } from "lucide-react";
import Link from "next/link";
import { useState, useEffect, useRef } from "react";
import { toast } from "sonner";

export interface LinkItem {
  id: number;
  short_code: string;
  short_url: string;
  original_url: string;
  title: string;
  description?: string;
  click_count: number;
  is_active: boolean;
  created_at: string;
  folder_id?: number;
  folder_name?: string;
  tags?: { id: number; name: string; color: string }[];
}

interface LinkCardProps {
  link: LinkItem;
  onDelete: (id: number) => void;
  workspaceSlug?: string;
}

export function LinkCard({ link, onDelete }: LinkCardProps) {
  const [copied, setCopied] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showQR, setShowQR] = useState(false);
  const [qrDataURL, setQrDataURL] = useState<string | null>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  const handleCopy = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    navigator.clipboard.writeText(link.short_url);
    setCopied(true);
    toast.success("已复制到剪贴板");
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDelete = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (deleting) return;
    setDeleting(true);
    try {
      await onDelete(link.id);
      toast.success("链接已删除");
    } catch {
      toast.error("删除失败");
      setDeleting(false);
    }
  };

  const handleShowQR = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setShowQR(true);
  };

  // Generate QR code when modal opens
  useEffect(() => {
    if (showQR && !qrDataURL) {
      import("qrcode").then((QRCode) => {
        const canvas = document.createElement("canvas");
        QRCode.toCanvas(canvas, link.short_url, {
          width: 256,
          margin: 2,
          color: { dark: "#000000", light: "#ffffff" },
          errorCorrectionLevel: "M",
        });
        setQrDataURL(canvas.toDataURL("image/png"));
      });
    }
  }, [showQR, qrDataURL, link.short_url]);

  return (
    <>
      <div className="group relative bg-white rounded-xl border border-gray-100 hover:border-gray-200 hover:shadow-md transition-all duration-200">
        <Link
          href={`/dashboard/links/${link.id}`}
          className="block p-4 sm:p-5"
        >
          <div className="flex items-start justify-between gap-4">
            {/* Left: Info */}
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <h3 className="font-medium text-gray-900 truncate text-sm sm:text-base">
                  {link.title || "未命名链接"}
                </h3>
                <span
                  className={`shrink-0 px-1.5 py-0.5 text-[10px] font-medium rounded-full ${
                    link.is_active
                      ? "bg-green-100 text-green-700"
                      : "bg-gray-100 text-gray-500"
                  }`}
                >
                  {link.is_active ? "启用" : "停用"}
                </span>
              </div>

              <div className="flex items-center gap-2 text-sm">
                <span className="text-indigo-600 font-mono text-xs sm:text-sm truncate">
                  {link.short_url}
                </span>
                <button
                  onClick={handleCopy}
                  className={`shrink-0 p-1 rounded transition-colors ${
                    copied
                      ? "text-green-500 bg-green-50"
                      : "text-gray-400 hover:text-indigo-500"
                  }`}
                  title="复制链接"
                >
                  {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                </button>
              </div>

              <p className="text-xs text-gray-400 mt-1 truncate max-w-md">
                {link.original_url}
              </p>

              {link.description && (
                <p className="text-xs text-gray-500 mt-1 line-clamp-1">{link.description}</p>
              )}
            </div>

            {/* Right: Stats & Actions */}
            <div className="flex items-center gap-3 shrink-0">
              <div className="hidden sm:flex items-center gap-1 text-sm">
                <BarChart3 className="w-3.5 h-3.5 text-gray-400" />
                <span className="font-semibold text-gray-700 tabular-nums">
                  {link.click_count.toLocaleString()}
                </span>
              </div>

              <button
                onClick={handleShowQR}
                className="p-1.5 text-gray-400 hover:text-indigo-500 hover:bg-indigo-50 rounded-lg transition-colors"
                title="二维码"
              >
                <QrCode className="w-4 h-4" />
              </button>

              <a
                href={link.short_url}
                target="_blank"
                onClick={(e) => e.stopPropagation()}
                className="p-1.5 text-gray-400 hover:text-indigo-500 hover:bg-indigo-50 rounded-lg transition-colors"
                title="打开链接"
              >
                <ExternalLink className="w-4 h-4" />
              </a>

              <Link
                href={`/dashboard/links/${link.id}`}
                onClick={(e) => e.stopPropagation()}
                className="p-1.5 text-gray-400 hover:text-indigo-500 hover:bg-indigo-50 rounded-lg transition-colors"
                title="编辑"
              >
                <Pencil className="w-4 h-4" />
              </Link>

              <button
                onClick={handleDelete}
                disabled={deleting}
                className="p-1.5 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors disabled:opacity-50"
                title="删除"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          </div>
        </Link>
      </div>

      {/* QR Code Modal */}
      {showQR && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
          onClick={() => setShowQR(false)}
        >
          <div
            className="bg-white rounded-2xl shadow-xl p-6 max-w-sm w-full mx-4"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold text-gray-900">二维码</h3>
              <button
                onClick={() => setShowQR(false)}
                className="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition"
              >
                <X className="size-4" />
              </button>
            </div>

            <div className="flex flex-col items-center gap-4">
              {/* QR Image */}
              <div className="bg-white border border-gray-100 rounded-xl p-3">
                {qrDataURL ? (
                  <img
                    src={qrDataURL}
                    alt="QR Code"
                    className="size-56"
                  />
                ) : (
                  <div className="size-56 flex items-center justify-center">
                    <div className="size-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
                  </div>
                )}
              </div>

              <p className="text-sm font-mono text-gray-600 text-center break-all">
                {link.short_url}
              </p>

              <div className="flex items-center gap-2 w-full">
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(link.short_url);
                    toast.success("已复制");
                  }}
                  className="flex-1 inline-flex items-center justify-center gap-1.5 rounded-lg border border-gray-200 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition"
                >
                  <Copy className="size-3.5" />
                  复制链接
                </button>
                {qrDataURL && (
                  <a
                    href={qrDataURL}
                    download={`kada-qr-${link.short_code}.png`}
                    className="inline-flex items-center justify-center gap-1.5 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 transition"
                  >
                    <Download className="size-3.5" />
                    下载
                  </a>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

/** Skeleton placeholder shown while loading */
export function LinkCardPlaceholder() {
  return (
    <div className="bg-white rounded-xl border border-gray-100 p-4 sm:p-5">
      <div className="flex items-start justify-between gap-4 animate-pulse">
        <div className="flex-1 space-y-2">
          <div className="flex items-center gap-2">
            <div className="h-5 w-32 bg-gray-200 rounded-md sm:w-44" />
            <div className="h-4 w-10 bg-gray-200 rounded-full" />
          </div>
          <div className="flex items-center gap-2">
            <div className="h-4 w-48 bg-gray-100 rounded-md" />
            <div className="h-3.5 w-3.5 bg-gray-100 rounded" />
          </div>
          <div className="h-3 w-64 bg-gray-100 rounded" />
        </div>
        <div className="flex items-center gap-3">
          <div className="hidden sm:block h-6 w-16 bg-gray-200 rounded-md" />
          <div className="h-8 w-8 bg-gray-100 rounded-lg" />
          <div className="h-8 w-8 bg-gray-100 rounded-lg" />
          <div className="h-8 w-8 bg-gray-100 rounded-lg" />
        </div>
      </div>
    </div>
  );
}
