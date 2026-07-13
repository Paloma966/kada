"use client";
import { MousePointerClick } from "lucide-react";
import { ComingSoon } from "@/components/ComingSoon";

export default function EventsPage() {
  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">事件</h1>
        <p className="text-sm text-gray-500 mt-1">追踪用户点击和转化事件</p>
      </div>
      <ComingSoon
        icon={MousePointerClick}
        title="事件追踪"
        description="即将支持转化事件追踪，了解每次点击背后的用户行为"
      />
    </div>
  );
}
