import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

const api = process.env.SQUIRRELBOT_API || 'http://localhost:8080';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			adapter: adapter({
				fallback: '200.html'
			})
		})
	],
	server: {
		proxy: {
			'/api': api,
			'/auth': api,
			'/healthz': api,
			'/readyz': api
		}
	}
});
