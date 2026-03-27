import { Variants } from "framer-motion";

// ──────────────────────────────────────────────────────────────
// Spring presets
// ──────────────────────────────────────────────────────────────
export const springPresets = {
  snappy: { type: "spring", stiffness: 400, damping: 30 },
  gentle: { type: "spring", stiffness: 300, damping: 35 },
  bouncy: { type: "spring", stiffness: 500, damping: 25 },
  smooth: { type: "spring", stiffness: 200, damping: 40 },
  inertia: { type: "spring", stiffness: 150, damping: 20 },
} as const;

// ──────────────────────────────────────────────────────────────
// Reusable animation variants
// ──────────────────────────────────────────────────────────────
export const fadeInUp: Variants = {
  hidden: { opacity: 0, y: 32 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { ...springPresets.gentle },
  },
};

export const fadeIn: Variants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1, transition: { duration: 0.5, ease: "easeOut" } },
};

export const scaleIn: Variants = {
  hidden: { opacity: 0, scale: 0.88 },
  visible: { opacity: 1, scale: 1, transition: { ...springPresets.bouncy } },
};

export const hoverLift: Variants = {
  rest: { y: 0, scale: 1, transition: springPresets.snappy },
  hover: { y: -4, scale: 1.02, transition: springPresets.snappy },
};

// ──────────────────────────────────────────────────────────────
// Stagger container / item
// ──────────────────────────────────────────────────────────────
export const staggerContainer: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.07,
      delayChildren: 0.1,
    },
  },
};

export const staggerItem: Variants = {
  hidden: { opacity: 0, y: 24 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { ...springPresets.gentle },
  },
};

// ──────────────────────────────────────────────────────────────
// Page transition
// ──────────────────────────────────────────────────────────────
export const pageTransition = {
  initial: { opacity: 0, y: 16 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -8 },
  transition: { ...springPresets.smooth },
};
