/**
 * Where the language capsule sits, shared by the three shells that show it.
 *
 * The app shell renders it inside its top bar, next to the theme toggle. The landing and sign-in pages have
 * no bar - they are dark in both themes, so neither shows a theme toggle either - and pin the capsule to
 * the viewport corner instead. All three have to land on the same pixel, or the capsule visibly jumps
 * sideways and up as you sign in, which is exactly what the hand-written copies of these numbers did (16px
 * against 24px on the horizontal, and four pixels higher on the dark pages).
 *
 * The pinned vertical offset is not a second rhythm: it is (56px bar - 32px control) / 2, the same centring
 * the bar does, with the bar removed. Change one of these and the other shell follows.
 */
export const TOP_BAR_HEIGHT = "h-14";

/** Horizontal padding of the bar. The pinned copy matches its right edge. */
export const TOP_BAR_GUTTER = "px-4 sm:px-6";

/**
 * The capsule sits one step past that gutter - 12px from the edge on phones, 20px from `sm` up.
 *
 * A pill is all curve, so on a straight margin it reads as inset: the eye measures from where the shape
 * looks solid, and a rounded end leaves a wedge of page beside the border. One 4px step buys that back. It
 * is taken here, once, rather than per page, because three shells hand-placing the same control is how the
 * copies drifted apart the first time.
 *
 * Only the pill. The theme toggle and the sign-in button are rounded rectangles whose straight sides have
 * no such wedge, and the title inside the bar keeps the gutter it shares with the content column below.
 * The pinned copy needs no second number: pinning the corner and letting the capsule overhang it lands on
 * the same pixel as the bar does.
 */
export const TOP_BAR_CAPSULE_OVERHANG = "-mr-1";

/** The same corner, for a page that has no bar to put the controls in. */
export const TOP_BAR_CONTROLS_PINNED = "fixed right-4 top-3 z-50 sm:right-6";
