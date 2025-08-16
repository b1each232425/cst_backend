/*
 * @Author: Mayux && dbs45412@163.com
 * @Date: 2025-08-16 15:50:47
 * @LastEditors: Mayux && dbs45412@163.com
 * @LastEditTime: 2025-08-16 17:14:09
 * @FilePath: \assess-db\assess\backend\tusd\web\vite.config.js
 * @Description: 
 * @Feature: 
 * @Dependencies: 
 * @Exported Methods: 
 * @Props: 
 * @Copyright: Copyright (c) 2025 by Mayux, All Rights Reserved. 
 */
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
				target: 'ws://localhost:6612/'
			},
			'/api': 'http://localhost:6612/',
		},
	},
});
