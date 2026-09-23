// The assistant answers in Markdown, but the chat bubble renders plain text, so
// every `**bold**` in an answer reached the user as literal asterisks. This
// module parses the small subset the model actually produces - bold, inline
// code, headings, bullets and numbered items - so the component can render it.
// A full Markdown dependency would be far more machinery than that subset needs,
// and this keeps everything as text: nothing here can turn into HTML.
//
// It is also written for a token stream. While an answer is still arriving, an
// unclosed `**` renders as bold to the end of what has arrived so far, instead
// of flashing a stray asterisk and repairing itself on the next chunk.

export type SpanKind = "text" | "bold" | "code";

export interface Span {
  kind: SpanKind;
  text: string;
}

export type LineKind = "text" | "bullet" | "ordered" | "heading";

export interface Line {
  kind: LineKind;
  /** Original marker of an ordered item ("3."); empty for every other kind. */
  marker: string;
  /** Empty for a blank line, which is what separates paragraphs. */
  spans: Span[];
}

const HEADING = /^#{1,6}\s+/;
const BULLET = /^\s*[-*•]\s+/;
const ORDERED = /^\s*(\d{1,2})[.)]\s+/;
const BOLD_MARKER = "**";
const CODE_MARKER = "`";

/** Join neighbouring plain-text spans, so callers see "a **b** c" as three. */
function mergeText(spans: Span[]): Span[] {
  return spans.reduce<Span[]>((out, span) => {
    const last = out[out.length - 1];
    if (last?.kind === "text" && span.kind === "text") {
      last.text += span.text;
      return out;
    }
    out.push({ ...span });
    return out;
  }, []);
}

/**
 * Split one line into styled spans.
 *
 * An unclosed `**` styles the rest of the line: mid-stream that is the correct
 * reading, and once the closing marker arrives it is parsed normally. An
 * unclosed backtick stays literal, because a lone backtick is more likely to be
 * text than the start of inline code.
 */
function parseSpans(text: string): Span[] {
  const spans: Span[] = [];
  let rest = text;

  while (rest) {
    const boldAt = rest.indexOf(BOLD_MARKER);
    const codeAt = rest.indexOf(CODE_MARKER);

    let at = -1;
    let kind: SpanKind = "text";
    let marker = "";
    if (boldAt >= 0 && (codeAt < 0 || boldAt < codeAt)) {
      at = boldAt;
      kind = "bold";
      marker = BOLD_MARKER;
    } else if (codeAt >= 0) {
      at = codeAt;
      kind = "code";
      marker = CODE_MARKER;
    }

    if (at < 0) {
      spans.push({ kind: "text", text: rest });
      break;
    }
    if (at > 0) {
      spans.push({ kind: "text", text: rest.slice(0, at) });
    }

    const after = rest.slice(at + marker.length);
    const close = after.indexOf(marker);
    if (close < 0) {
      if (kind === "bold" && after) {
        spans.push({ kind: "bold", text: after });
      } else {
        // Nothing to style: keep the marker itself rather than dropping it.
        spans.push({ kind: "text", text: marker + after });
      }
      break;
    }

    spans.push({ kind, text: after.slice(0, close) });
    rest = after.slice(close + marker.length);
  }

  return mergeText(spans);
}

/** Split an answer into one entry per line, blanks included. */
export function parseAnswer(text: string): Line[] {
  return text.split("\n").map((raw) => {
    const line = raw.trimEnd();

    const heading = line.match(HEADING);
    if (heading) {
      return { kind: "heading", marker: "", spans: parseSpans(line.slice(heading[0].length)) };
    }

    const ordered = line.match(ORDERED);
    if (ordered) {
      return {
        kind: "ordered",
        marker: `${ordered[1]}.`,
        spans: parseSpans(line.slice(ordered[0].length)),
      };
    }

    const bullet = line.match(BULLET);
    if (bullet) {
      return { kind: "bullet", marker: "", spans: parseSpans(line.slice(bullet[0].length)) };
    }

    return { kind: "text", marker: "", spans: parseSpans(line) };
  });
}
