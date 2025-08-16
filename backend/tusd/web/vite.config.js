import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		//--max-http-header-size=1048576
		host: '0.0.0.0',
		port: 8080,
		// open: true,
		// https: false,
		proxy: {

			'/api/ws': {
				ws: true,
				target: 'ws://localhost:6616/'
			},
			'/api': 'http://localhost:6616/',
		},
	},
});
