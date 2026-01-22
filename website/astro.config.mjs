// @ts-check
import { defineConfig } from 'astro/config';

import vue from '@astrojs/vue';
import sitemap from '@astrojs/sitemap';
import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
  integrations: [vue(), sitemap()],
  site: 'https://felixgeelhaar.github.io',
  base: process.env.NODE_ENV === 'development' ? '/' : '/simon',
  output: 'static',
  vite: {
    plugins: [tailwindcss()]
  }
});
