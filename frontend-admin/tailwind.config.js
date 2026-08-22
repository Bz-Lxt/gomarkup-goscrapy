/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  corePlugins: {
    preflight: false,
  },
  theme: {
    extend: {
      colors: {
        bg0: '#071018',
        bg1: '#0C1A24',
        bg2: '#122632',
        line: '#1E3A47',
        cyan: '#3EE0C5',
        amber: '#F5B942',
        rose: '#FF6B7A',
        ink: '#E7F4F2',
        muted: '#7FA3AE',
      },
      fontFamily: {
        display: ['Syne', 'sans-serif'],
        sans: ['IBM Plex Sans', 'sans-serif'],
        mono: ['IBM Plex Mono', 'monospace'],
      },
      screens: {
        phone: { max: '479px' },
        tablet: { max: '767px' },
      },
    },
  },
  plugins: [],
}
