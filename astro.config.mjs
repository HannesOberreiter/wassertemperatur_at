import { defineConfig } from "astro/config";
import tailwind from "@astrojs/tailwind";
import alpinejs from "@astrojs/alpinejs";

import vercel from "@astrojs/vercel/serverless";

// https://astro.build/config
export default defineConfig({
  output: "server",
  integrations: [
    tailwind({
      applyBaseStyles: false,
    }),
    alpinejs(),
  ],
  adapter: vercel(),
});
