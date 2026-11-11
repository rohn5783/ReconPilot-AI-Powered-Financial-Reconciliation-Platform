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
        ledger: {
          950: '#06070A',
          900: '#090A0F', // Abyssal canvas
          850: '#0E1017', // Surface / Card
          800: '#141824', // Elevated panel / hover
          750: '#191E2D', // Selected state
          700: '#1E2433', // Border / Divider
          600: '#2A3245', // Active border
          400: '#8E9BB0', // Secondary text
          300: '#BAC4D6', // Primary muted
          100: '#F3F6FB', // High contrast text
        },
        recon: {
          emerald: '#10B981', // Reconciled / Match
          emeraldGlow: 'rgba(16, 185, 129, 0.15)',
          crimson: '#F43F5E', // Discrepancy / Overcharge
          crimsonGlow: 'rgba(244, 63, 94, 0.15)',
          amber: '#F59E0B',   // In dispute / Warning
          amberGlow: 'rgba(245, 158, 11, 0.15)',
          cyan: '#06B6D4',    // Bank statement
          indigo: '#6366F1',  // AI Synthesis / Gateway
        }
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'SF Mono', 'Menlo', 'Consolas', 'monospace'],
      },
      animation: {
        'pulse-subtle': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'fade-in': 'fadeIn 0.2s cubic-bezier(0.16, 1, 0.3, 1)',
        'slide-in': 'slideIn 0.25s cubic-bezier(0.16, 1, 0.3, 1)',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideIn: {
          '0%': { transform: 'translateX(100%)' },
          '100%': { transform: 'translateX(0)' },
        },
      },
    },
  },
  plugins: [],
}

