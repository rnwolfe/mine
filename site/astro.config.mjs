import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import rehypeMermaid from 'rehype-mermaid';

// https://astro.build/config
export default defineConfig({
  site: 'https://mine.rwolfe.io',
  integrations: [
    starlight({
      title: 'mine',
      description: 'Your personal developer supercharger',
      logo: {
        src: './src/assets/logo.svg',
        replacesTitle: true,
      },
      social: {
        github: 'https://github.com/rnwolfe/mine',
      },
      customCss: [
        './src/styles/custom.css',
      ],
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Installation', slug: 'getting-started/installation' },
            { label: 'Quick Start', slug: 'getting-started/quick-start' },
          ],
        },
        {
          label: 'Commands',
          autogenerate: { directory: 'commands' },
        },
        {
          label: 'Contributing',
          items: [
            { label: 'Architecture', slug: 'contributing/architecture' },
            { label: 'Plugin Protocol', slug: 'contributing/plugin-protocol' },
          ],
        },
      ],
      components: {
        ThemeSelect: './src/components/ThemeSelect.astro',
      },
    }),
  ],
  markdown: {
    rehypePlugins: [
      [rehypeMermaid, { strategy: 'inline-svg', dark: true }],
    ],
  },
});
