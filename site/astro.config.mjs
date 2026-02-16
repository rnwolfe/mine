import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightClientMermaid from '@pasqal-io/starlight-client-mermaid';

// https://astro.build/config
export default defineConfig({
	site: 'https://mine.rwolfe.io',
	integrations: [
		starlight({
			title: 'mine',
			description: 'Your personal developer supercharger',
			logo: {
				light: './src/assets/logo-light.svg',
				dark: './src/assets/logo-dark.svg',
				replacesTitle: false,
			},
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/rnwolfe/mine' },
			],
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
					label: 'For Contributors',
					items: [
						{ label: 'Architecture', slug: 'contributors/architecture' },
						{ label: 'Plugin Protocol', slug: 'contributors/plugin-protocol' },
					],
				},
			],
			plugins: [
				starlightClientMermaid(),
			],
		}),
	],
});
