/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class', // Enable class-based dark mode
  theme: {
    extend: {
      colors: {
        // Custom theme colors for better dark mode support
        primary: {
          light: '#3b82f6', // blue-500
          dark: '#60a5fa',  // blue-400
        },
        background: {
          light: '#f9fafb', // gray-50
          dark: '#111827',  // gray-900
        },
        surface: {
          light: '#ffffff',
          dark: '#1f2937',  // gray-800
        },
        border: {
          light: '#e5e7eb', // gray-200
          dark: '#374151',  // gray-700
        },
        // Ocean theme
        ocean: {
          50: '#f0f9ff',
          100: '#e0f2fe',
          200: '#bae6fd',
          300: '#7dd3fc',
          400: '#38bdf8',
          500: '#0ea5e9',
          600: '#0284c7',
          700: '#0369a1',
          800: '#075985',
          900: '#0c4a6e',
          950: '#082f49',
        },
        teal: {
          50: '#f0fdfa',
          100: '#ccfbf1',
          200: '#99f6e4',
          300: '#5eead4',
          400: '#2dd4bf',
          500: '#14b8a6',
          600: '#0d9488',
          700: '#0f766e',
          800: '#115e59',
          900: '#134e4a',
          950: '#042f2e',
        },
        // Forest theme
        forest: {
          50: '#f0fdf4',
          100: '#dcfce7',
          200: '#bbf7d0',
          300: '#86efac',
          400: '#4ade80',
          500: '#22c55e',
          600: '#16a34a',
          700: '#15803d',
          800: '#166534',
          900: '#14532d',
          950: '#052e16',
        },
        brown: {
          50: '#fdf8f6',
          100: '#f2e8e5',
          200: '#eaddd7',
          300: '#e0cec7',
          400: '#d2bab0',
          500: '#bfa094',
          600: '#a18072',
          700: '#977669',
          800: '#846358',
          900: '#43302b',
          950: '#27191a',
        },
        // Sunset theme
        sunset: {
          50: '#fff7ed',
          100: '#ffedd5',
          200: '#fed7aa',
          300: '#fdba74',
          400: '#fb923c',
          500: '#f97316',
          600: '#ea580c',
          700: '#c2410c',
          800: '#9a3412',
          900: '#7c2d12',
          950: '#431407',
        },
        purple: {
          50: '#faf5ff',
          100: '#f3e8ff',
          200: '#e9d5ff',
          300: '#d8b4fe',
          400: '#c084fc',
          500: '#a855f7',
          600: '#9333ea',
          700: '#7e22ce',
          800: '#6b21a8',
          900: '#581c87',
          950: '#3b0764',
        },
        // Melodee brand colors (from logo)
        melodee: {
          50: '#fef1f7',
          100: '#fee5f0',
          200: '#ffcce3',
          300: '#ffa3cd',
          400: '#ff6baa',
          500: '#f93a8a',
          600: '#e61866',
          700: '#c7104b',
          800: '#a5103f',
          900: '#8a1138',
          950: '#54051d',
        },
        coral: {
          50: '#fff4ed',
          100: '#ffe6d5',
          200: '#fecaaa',
          300: '#fda574',
          400: '#fb753c',
          500: '#f95016',
          600: '#ea360c',
          700: '#c2250c',
          800: '#9a2012',
          900: '#7c1d12',
          950: '#430a07',
        },
      },
    },
  },
  plugins: [],
}