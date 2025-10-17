// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightThemeRapide from 'starlight-theme-rapide';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'compak',
			description: 'A package manager for Docker Compose applications',
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/LoriKarikari/compak' }
			],
			plugins: [starlightThemeRapide()],
			sidebar: [
				{
					label: 'Getting Started',
					items: [
						{ label: 'Introduction', slug: 'index' },
						{ label: 'Installation', slug: 'getting-started/installation' },
						{ label: 'Quick Start', slug: 'getting-started/quick-start' },
					],
				},
				{
					label: 'Guides',
					items: [
						{ label: 'Installing Packages', slug: 'guides/installing' },
						{ label: 'Version Management', slug: 'guides/versioning' },
						{ label: 'Publishing Packages', slug: 'guides/publishing' },
						{ label: 'Creating Packages', slug: 'guides/creating' },
					],
				},
				{
					label: 'CLI Commands',
					collapsed: true,
					items: [
						{ label: 'Overview', slug: 'reference/cli' },
						{ label: 'install', slug: 'reference/commands/install' },
						{ label: 'upgrade', slug: 'reference/commands/upgrade' },
						{ label: 'uninstall', slug: 'reference/commands/uninstall' },
						{ label: 'list', slug: 'reference/commands/list' },
						{ label: 'status', slug: 'reference/commands/status' },
						{ label: 'search', slug: 'reference/commands/search' },
						{ label: 'update', slug: 'reference/commands/update' },
						{ label: 'extract', slug: 'reference/commands/extract' },
						{ label: 'publish', slug: 'reference/commands/publish' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'Package Format', slug: 'reference/package-format' },
						{ label: 'Environment Variables', slug: 'reference/environment' },
					],
				},
			],
		}),
	],
});
