/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./internal/views/**/*.templ",
    "./internal/views/**/*_templ.go"
  ],
  theme: {
    extend: {
      colors: {
        clay: {
          50: "#faf7f2",
          100: "#efe7da",
          200: "#dfd1bc",
          800: "#4e4032",
          900: "#2f261e"
        }
      },
      backgroundImage: {
        shell: "radial-gradient(circle at 0% 0%, rgba(208, 178, 143, 0.25), transparent 40%), radial-gradient(circle at 100% 20%, rgba(125, 158, 106, 0.18), transparent 35%), linear-gradient(180deg, #f8fafc 0%, #f5f7fb 100%)"
      },
      fontFamily: {
        sans: ["Manrope", "ui-sans-serif", "system-ui", "sans-serif"]
      }
    }
  },
  plugins: []
};
