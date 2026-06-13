/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        background: 'var(--background)',
        foreground: 'var(--foreground)',
        panel: {
          DEFAULT: 'var(--panel-bg)',
          hover: 'var(--panel-bg-hover)',
          border: 'var(--panel-border)',
          'border-hover': 'var(--panel-border-hover)',
        },
        brand: {
          yellow: 'var(--brand-yellow)',
          black: 'var(--brand-black)',
          'dark-gray': 'var(--brand-dark-gray)',
          'light-gray': 'var(--brand-light-gray)',
        },
        text: {
          primary: 'var(--text-primary)',
          secondary: 'var(--text-secondary)',
          tertiary: 'var(--text-tertiary)',
          disabled: 'var(--text-disabled)',
        },
        binance: {
          green: 'var(--binance-green)',
          red: 'var(--binance-red)',
          yellow: 'var(--binance-yellow)',
        },
      },
    },
  },
  plugins: [],
}
