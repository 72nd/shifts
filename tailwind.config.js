// Tailwind config for the standalone CLI (see `task css`). Only the content
// globs matter: daisyUI is vendored as plain CSS (static/vendor/daisyui.min.css).
module.exports = {
  content: [
    "./internal/web/templates/**/*.html",
    "./internal/web/static/js/**/*.js",
  ],
  theme: {
    extend: {},
  },
};
