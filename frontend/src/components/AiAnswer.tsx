import type { Span } from "@/lib/aiAnswer";
import { parseAnswer } from "@/lib/aiAnswer";

function Inline({ spans }: { spans: Span[] }) {
  return (
    <>
      {spans.map((span, i) => {
        if (span.kind === "bold") {
          return (
            <strong key={i} className="font-semibold">
              {span.text}
            </strong>
          );
        }
        if (span.kind === "code") {
          return (
            <code
              key={i}
              className="rounded bg-canvas px-1 py-0.5 font-mono text-[0.85em]"
            >
              {span.text}
            </code>
          );
        }
        return <span key={i}>{span.text}</span>;
      })}
    </>
  );
}

/**
 * Renders one assistant answer.
 *
 * Only the assistant's text goes through here: what the user typed is shown
 * verbatim. See `lib/aiAnswer.ts` for what is parsed and why.
 */
export function AiAnswer({ text }: { text: string }) {
  return (
    <>
      {parseAnswer(text).map((line, i) => {
        if (line.spans.length === 0) {
          return <div key={i} className="h-2" />;
        }
        if (line.kind === "bullet") {
          return (
            <div key={i} className="flex gap-1.5">
              <span aria-hidden className="text-faint">
                •
              </span>
              <span className="min-w-0 flex-1">
                <Inline spans={line.spans} />
              </span>
            </div>
          );
        }
        if (line.kind === "ordered") {
          return (
            <div key={i} className="flex gap-1.5">
              <span className="text-muted tabular-nums">{line.marker}</span>
              <span className="min-w-0 flex-1">
                <Inline spans={line.spans} />
              </span>
            </div>
          );
        }
        return (
          <div key={i} className={line.kind === "heading" ? "font-semibold" : undefined}>
            <Inline spans={line.spans} />
          </div>
        );
      })}
    </>
  );
}
