import type { Locale } from "@/lib/i18n";

/**
 * Flags for the language switcher, drawn as inline SVG.
 *
 * Emoji flags (🇨🇳 / 🇬🇧) are not used on purpose: Windows has no glyphs for regional-indicator pairs,
 * so Chrome and Edge there render "CN" / "GB" as letters instead of a flag - which is exactly the
 * "ugly text" this is meant to replace. Inline SVG looks the same everywhere, needs no extra request
 * and inherits the button's size.
 *
 * Simplified deliberately: these are 20px wide marks, not accurate flags. Details that would be noise
 * at this size (the stars' positions, the Union Jack's diagonals) are approximated so the shape still
 * reads as the right flag.
 */
export function Flag({ locale, className = "size-4" }: { locale: Locale; className?: string }) {
  return locale === "zh" ? <ChinaFlag className={className} /> : <EnglishFlag className={className} />;
}

/** China: red field with a large star and four smaller ones. */
function ChinaFlag({ className }: { className: string }) {
  return (
    <svg viewBox="0 0 30 20" className={className} role="presentation" focusable="false">
      <rect width="30" height="20" fill="#de2910" />
      <g fill="#ffde00">
        <Star cx={6} cy={5} r={3} />
        <Star cx={12} cy={2} r={1} />
        <Star cx={12} cy={5} r={1} />
        <Star cx={12} cy={8} r={1} />
        <Star cx={9.5} cy={10.5} r={1} />
      </g>
    </svg>
  );
}

/** A five-pointed star centred on (cx, cy), with `r` as its outer radius. */
function Star({ cx, cy, r }: { cx: number; cy: number; r: number }) {
  const points: string[] = [];
  for (let i = 0; i < 10; i++) {
    // Alternate between the outer radius and the inner radius, starting at the top (-90 degrees).
    const radius = i % 2 === 0 ? r : r * 0.382;
    const angle = (Math.PI / 5) * i - Math.PI / 2;
    points.push(`${(cx + radius * Math.cos(angle)).toFixed(2)},${(cy + radius * Math.sin(angle)).toFixed(2)}`);
  }
  return <polygon points={points.join(" ")} />;
}

/**
 * English: the Union Jack.
 *
 * A neutral stand-in for a language rather than a country - English is not owned by the United Kingdom
 * - but it is the mark readers expect on an EN/中文 toggle rather than a US or UK flag alone.
 */
function EnglishFlag({ className }: { className: string }) {
  return (
    <svg viewBox="0 0 30 20" className={className} role="presentation" focusable="false">
      <rect width="30" height="20" fill="#012169" />
      {/* White diagonals first, then the narrower coloured ones on top of them. */}
      <path d="M0 0 L30 20 M30 0 L0 20" stroke="#ffffff" strokeWidth="4" />
      <path d="M0 0 L30 20 M30 0 L0 20" stroke="#c8102e" strokeWidth="2" />
      <path d="M15 0 V20 M0 10 H30" stroke="#ffffff" strokeWidth="6.6" />
      <path d="M15 0 V20 M0 10 H30" stroke="#c8102e" strokeWidth="4" />
    </svg>
  );
}
