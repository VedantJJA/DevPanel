import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) => filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter({
				pages: 'build',
				assets: 'build',
				fallback: '200.html',  // SPA fallback for client-side routing
				precompress: false,
				strict: false
			})
		})
	],
	server: {
		proxy: {
			// Proxy API calls to the Go backend during dev
			'/api': 'http://localhost:8090',
			'/healthz': 'http://localhost:8090',
			'/ws': {
				target: 'http://localhost:8090',
				ws: true
			}
		}
	}
});

