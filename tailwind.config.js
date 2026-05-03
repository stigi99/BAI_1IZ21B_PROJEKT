/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./internal/views/**/*.templ",
    "./internal/views/**/*_templ.go"
  ],
  theme: {
    extend: {
      colors: {
        sakura: {
          50: "#fff1fb",
          100: "#ffd9f3",
          200: "#ffb7e7",
          500: "#ec4899",
          700: "#be185d",
          900: "#831843"
        },
        neon: {
          50: "#ecfeff",
          100: "#cffafe",
          200: "#a5f3fc",
          500: "#06b6d4",
          700: "#0e7490",
          900: "#164e63"
        },
        clay: {
          50: "#faf7f2",
          100: "#efe7da",
          200: "#dfd1bc",
          800: "#4e4032",
          900: "#2f261e"
        }
      },
      backgroundImage: {
        shell: "radial-gradient(circle at 12% 18%, rgba(236, 72, 153, 0.22), transparent 28%), radial-gradient(circle at 88% 12%, rgba(6, 182, 212, 0.22), transparent 24%), radial-gradient(circle at 50% 90%, rgba(168, 85, 247, 0.18), transparent 28%), linear-gradient(180deg, #fff7fb 0%, #f5fbff 46%, #fdf2f8 100%)"
      },
      fontFamily: {
        sans: ["M PLUS Rounded 1c", "Manrope", "ui-sans-serif", "system-ui", "sans-serif"],
        mono: ["JetBrains Mono", "ui-monospace", "monospace"]
      }
    }
  },
  plugins: []
};
