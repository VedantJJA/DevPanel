import { fetchConfig } from '../runtime/config';

// Enable SPA mode: use client-side routing for dynamic parameterized routes
export const prerender = false;
export const ssr = false;

export async function load() {
	const config = await fetchConfig();
	return { config };
}
