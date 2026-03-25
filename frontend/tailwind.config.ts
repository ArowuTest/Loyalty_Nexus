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
          400: "#f9c74f",
          500: "#f3a53a",
          600: "#e07d25",
        },
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "sans-serif"],
        display: ["Syne", "Inter", "sans-serif"],
      },
      animation: {
        "spin-wheel": "spin 3s cubic-bezier(0.32, 1.0, 0.75, 1.0) forwards",
        "pulse-glow": "pulseGlow 2s infinite",
        "float": "float 3s ease-in-out infinite",
        "slide-up": "slideUp 0.3s ease-out",
      },
      keyframes: {
        pulseGlow: {
          "0%, 100%": { boxShadow: "0 0 0 0 rgba(95, 114, 249, 0.4)" },
          "50%": { boxShadow: "0 0 20px 8px rgba(95, 114, 249, 0.2)" },
        },
        float: {
          "0%, 100%": { transform: "translateY(0)" },
          "50%": { transform: "translateY(-8px)" },
        },
        slideUp: {
          from: { opacity: "0", transform: "translateY(12px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
      },
      backgroundImage: {
        "nexus-gradient": "linear-gradient(135deg, #5f72f9 0%, #8b5cf6 50%, #f9c74f 100%)",
        "card-gradient": "linear-gradient(145deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)",
      },
    },
  },
  plugins: [forms],
};

export default config;
