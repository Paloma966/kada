/**
 * Where the theme and language controls sit, shared by the two shells that show them.
 *
 * The app shell renders them inside its top bar. The sign-in page has no bar - it is a centred card on a
 * dark page - so it pins the same two controls to the viewport corner instead. Both have to land on the
 * same pixel, or the language capsule visibly jumps sideways and up as you sign in, which is exactly what
 * the two hand-written copies of these numbers did (16px against 24px on the horizontal, and four pixels
 * higher on the dark pages).
 *
 * The pinned vertical offset is not a second rhythm: it is (56px bar - 32px control) / 2, the same centring
 * the bar does, with the bar removed. Change one of these and the other shell follows.
 */
export const TOP_BAR_HEIGHT = "h-14";

/** Horizontal padding of the bar. The pinned copy matches its right edge. */
export const TOP_BAR_GUTTER = "px-4 sm:px-6";

/** The same corner, for a page that has no bar to put the controls in. */
export const TOP_BAR_CONTROLS_PINNED = "fixed right-4 top-3 z-50 sm:right-6";
