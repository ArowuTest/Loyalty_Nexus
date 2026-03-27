import type { Config } from "tailwindcss";
import forms from "@tailwindcss/forms";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        nexus: {
          50:  "#f0f4ff",
          100: "#dfe8ff",
          200: "#c2d3ff",
          300: "#9cb7ff",
          400: "#7d96ff",
          500: "#5f72f9",
          600: "#4a56ee",
          700: "#3c45d3",
          800: "#313aaa",
          900: "#2d3687",
        },
        gold: {
          50:  "#fffbeb",
          100: "#fef3c7",
          200: "#fde68a",
          300: "#fcd34d",
          400: "#FFE066",
          500: "#F5A623",
          600: "#F59E0B",
          700: "#d97706",
          800: "#b45309",
          900: "#92400e",
        },
        surface: {
          0: "#0d0e14",
          1: "#111219",
          2: "#171921",
          3: "#1e2030",
        },
        cyan: {
          DEFAULT: "#00D4FF",
          dark:    "#00B8E0",
        },
      },
      fontFamily: {
        sans:    ["Inter", "system-ui", "sans-serif"],
        display: ["Syne", "Inter", "sans-serif"],
      },
      animation: {
        "spin-wheel":   "spin 3s cubic-bezier(0.32, 1.0, 0.75, 1.0) forwards",
        "pulse-glow":   "pulseGlow 2s infinite",
        "float":        "float-y 4s ease-in-out infinite",
        "float-slow":   "float-y-slow 6s ease-in-out infinite",
        "slide-up":     "slideUp 0.3s ease-out",
        "aurora-1":     "aurora-1 14s ease-in-out infinite",
        "aurora-2":     "aurora-2 18s ease-in-out infinite",
        "aurora-3":     "aurora-3 22s ease-in-out infinite",
        "ticker":       "ticker-left 32s linear infinite",
        "breathe":      "breathe 3.5s ease-in-out infinite",
        "spin-slow":    "spin-very-slow 30s linear infinite",
        "pulse-ring":   "pulse-ring 2.2s ease-out infinite",
        "shimmer":      "shimmer-sweep 3s linear infinite",
        "gradient-x":   "gradient-x 5s ease infinite",
      },
      keyframes: {
        pulseGlow: {
          "0%, 100%": { boxShadow: "0 0 0 0 rgba(95, 114, 249, 0.4)" },
          "50%":      { boxShadow: "0 0 20px 8px rgba(95, 114, 249, 0.2)" },
        },
        "float-y": {
          "0%, 100%": { transform: "translateY(0px)" },
          "50%":      { transform: "translateY(-10px)" },
        },
        "float-y-slow": {
          "0%, 100%": { transform: "translateY(0px)" },
          "50%":      { transform: "translateY(-6px)" },
        },
        slideUp: {
          from: { opacity: "0", transform: "translateY(12px)" },
          to:   { opacity: "1", transform: "translateY(0)" },
        },
        "aurora-1": {
          "0%,100%": { transform: "translate(0%,0%) scale(1)",    opacity: "0.55" },
          "33%":     { transform: "translate(4%,-6%) scale(1.08)", opacity: "0.70" },
          "66%":     { transform: "translate(-3%,4%) scale(0.95)", opacity: "0.50" },
        },
        "aurora-2": {
          "0%,100%": { transform: "translate(0%,0%) scale(1.05)",  opacity: "0.45" },
          "40%":     { transform: "translate(-5%,5%) scale(0.92)", opacity: "0.65" },
          "70%":     { transform: "translate(6%,-4%) scale(1.10)", opacity: "0.40" },
        },
        "aurora-3": {
          "0%,100%": { transform: "translate(0%,0%) scale(1)",    opacity: "0.35" },
          "50%":     { transform: "translate(3%,-8%) scale(1.12)", opacity: "0.55" },
        },
        "ticker-left": {
          "0%":   { transform: "translateX(0)" },
          "100%": { transform: "translateX(-50%)" },
        },
        "spin-very-slow": {
          from: { transform: "rotate(0deg)" },
          to:   { transform: "rotate(360deg)" },
        },
        breathe: {
          "0%,100%": { transform: "scale(1)" },
          "50%":     { transform: "scale(1.03)" },
        },
        "pulse-ring": {
          "0%":   { transform: "scale(0.95)", boxShadow: "0 0 0 0 rgba(245,166,35,0.5)" },
          "70%":  { transform: "scale(1)",    boxShadow: "0 0 0 18px rgba(245,166,35,0)" },
          "100%": { transform: "scale(0.95)", boxShadow: "0 0 0 0 rgba(245,166,35,0)" },
        },
        "shimmer-sweep": {
          "0%":   { backgroundPosition: "-200% center" },
          "100%": { backgroundPosition: "200% center" },
        },
        "gradient-x": {
          "0%,100%": { backgroundPosition: "0% 50%" },
          "50%":     { backgroundPosition: "100% 50%" },
        },
      },
      backgroundImage: {
        "nexus-gradient":  "linear-gradient(135deg, #5f72f9 0%, #8b5cf6 50%, #f9c74f 100%)",
        "card-gradient":   "linear-gradient(145deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)",
        "gold-gradient":   "linear-gradient(135deg, #F5A623, #FFE066, #F59E0B)",
        "hero-radial":     "radial-gradient(ellipse 80% 50% at 50% -10%, rgba(245,166,35,0.15), transparent)",
        "wars-gradient":   "linear-gradient(135deg, rgba(245,166,35,0.08) 0%, rgba(0,212,255,0.05) 100%)",
      },
      boxShadow: {
        "gold-glow":    "0 0 20px rgba(245,166,35,0.35), 0 0 60px rgba(245,166,35,0.12)",
        "gold-glow-sm": "0 0 10px rgba(245,166,35,0.30)",
        "cyan-glow":    "0 0 20px rgba(0,212,255,0.30), 0 0 60px rgba(0,212,255,0.10)",
        "card":         "0 4px 24px rgba(0,0,0,0.40)",
        "card-hover":   "0 8px 40px rgba(0,0,0,0.55)",
      },
      borderRadius: {
        "2xl": "1rem",
        "3xl": "1.25rem",
        "4xl": "1.5rem",
      },
    },
  },
  plugins: [forms],
};

export default config;
