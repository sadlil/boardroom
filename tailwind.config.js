/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./ui/**/*.html",
    "./ui/**/*.go",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
      },
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
