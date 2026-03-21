/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        shell: {
          bg: "#1a1b26",
          panel: "#16161e",
          border: "#2d2f3a",
          accent: "#7aa2f7",
          muted: "#565f89",
        },
      },
    },
  },
  plugins: [],
};
