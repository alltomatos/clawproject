/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        apple: {
          blue: '#007AFF',
          gray: '#8E8E93',
          bg: '#F2F2F7',
          card: '#FFFFFF'
        }
      },
      borderRadius: {
        'apple': '24px',
      }
    },
  },
  plugins: [],
}
