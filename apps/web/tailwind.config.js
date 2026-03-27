/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        sand: "#f4efe6",
        ink: "#18222d",
        coral: "#da6a4e",
        pine: "#2d5a4a",
        mist: "#d7e1dd"
      },
      fontFamily: {
        sans: ["'Inter'", "'DM Sans'", "'Noto Sans TC'", "sans-serif"],
        display: ["'Space Grotesk'", "'Noto Sans TC'", "sans-serif"]
      },
      boxShadow: {
        card: "0 18px 60px rgba(24, 34, 45, 0.10)"
      }
    }
  },
  plugins: []
};
