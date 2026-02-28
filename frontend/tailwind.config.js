/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        background: '#09090b',
        card: '#18181b',
        border: '#27272a',
        primary: {
          DEFAULT: '#22c55e',
          foreground: '#09090b',
        },
      },
      borderRadius: {
        '4xl': '2rem',
        '3xl': '1.5rem',
      },
    },
  },
  plugins: [],
}
