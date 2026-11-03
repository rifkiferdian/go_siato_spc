import { cp, mkdir } from "node:fs/promises";

const copies = [
  ["node_modules/@fontsource/roboto/files/roboto-latin-400-normal.woff2", "web/static/vendor/roboto/roboto-latin-400-normal.woff2"],
  ["node_modules/@fontsource/roboto/files/roboto-latin-500-normal.woff2", "web/static/vendor/roboto/roboto-latin-500-normal.woff2"],
  ["node_modules/@fontsource/roboto/files/roboto-latin-700-normal.woff2", "web/static/vendor/roboto/roboto-latin-700-normal.woff2"],
  ["node_modules/@fontsource/roboto/files/roboto-latin-900-normal.woff2", "web/static/vendor/roboto/roboto-latin-900-normal.woff2"],
  ["node_modules/@fortawesome/fontawesome-free/css/all.min.css", "web/static/vendor/fontawesome/css/all.min.css"],
  ["node_modules/@fortawesome/fontawesome-free/webfonts", "web/static/vendor/fontawesome/webfonts"],
  ["node_modules/sweetalert2/dist/sweetalert2.min.css", "web/static/vendor/sweetalert2/sweetalert2.min.css"],
  ["node_modules/sweetalert2/dist/sweetalert2.all.min.js", "web/static/vendor/sweetalert2/sweetalert2.all.min.js"]
];

for (const [source, destination] of copies) {
  await mkdir(destination.substring(0, destination.lastIndexOf("/")), { recursive: true });
  await cp(source, destination, { recursive: true, force: true });
}

console.log("Roboto, Font Awesome, dan SweetAlert2 disalin ke web/static/vendor.");
