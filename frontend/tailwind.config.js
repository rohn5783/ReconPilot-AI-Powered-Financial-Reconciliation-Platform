/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Swiss Industrial Obsidian & Carbon Surface Palette
        carbon: {
          canvas: '#08090C',       // Primary background
          panel: '#0D0F14',        // Sidebars & elevated split panes
          card: '#10121A',         // Secondary surfaces
          hover: '#141824',        // Interactive hover states
          active: '#1A2030',       // Active/selected rows
          border: '#1E222D',       // Hairline structural dividers
          borderSubtle: '#161922', // Subtle sub-dividers
        },
        // Muted High-Contrast Signal Palette
        signal: {
          critical: '#F43F5E',
          criticalBg: '#2A0C14',
          high: '#F97316',
          highBg: '#2E1405',
          medium: '#EAB308',
          mediumBg: '#2A2004',
          nominal: '#10B981',
          nominalBg: '#06281E',
          accent: '#38BDF8',
          accentBg: '#082F49',
        },
        typography: {
          primary: '#E2E8F0',
          secondary: '#94A3B8',
          muted: '#64748B',
          dim: '#475569',
        },
      },
      fontFamily: {
        sans: ['Inter', 'Geist Sans', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['JetBrains Mono', 'Geist Mono', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
      borderRadius: {
        none: '0px',
        micro: '2px',
        sm: '4px',
        DEFAULT: '4px',
        md: '4px',
      },
      letterSpacing: {
        tightest: '-0.025em',
        tight: '-0.015em',
      },
    },
  },
  plugins: [],
};

