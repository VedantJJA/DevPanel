// Shared routing configuration store.
// Reads routing_mode ('path' | 'subdomain') and base_domain from /api/settings
// and provides helpers to generate project/service URLs for both modes.
import { writable, get } from 'svelte/store';

export interface RoutingConfig {
	mode: 'path' | 'subdomain';
	baseDomain: string;
}

export const routingConfig = writable<RoutingConfig>({
	mode: 'path',
	baseDomain: 'localhost:8090'
});

let loaded = false;

export async function loadRoutingConfig(): Promise<void> {
	try {
		const res = await fetch('/api/settings');
		if (res.ok) {
			const s = await res.json();
			routingConfig.set({
				mode: s.routing_mode === 'subdomain' ? 'subdomain' : 'path',
				baseDomain: s.base_domain || 'localhost:8090'
			});
			loaded = true;
		}
	} catch (e) {
		console.error('Failed to load routing config:', e);
	}
}

/** Returns the public URL for a project's primary (frontend/web) service. */
export function getProjectUrl(projectName: string, config?: RoutingConfig): string {
	const cfg = config ?? get(routingConfig);
	if (cfg.mode === 'subdomain') {
		const domain = cfg.baseDomain;
		const scheme = domain.startsWith('localhost') || /^127\./.test(domain) ? 'http' : 'https';
		return `${scheme}://${projectName}.${domain}/`;
	}
	return `/app/${encodeURIComponent(projectName)}/`;
}

/** Returns the public URL for a specific service within a project. */
export function getServiceUrl(
	projectName: string,
	serviceName: string,
	config?: RoutingConfig
): string {
	const cfg = config ?? get(routingConfig);
	if (cfg.mode === 'subdomain') {
		const domain = cfg.baseDomain;
		const scheme = domain.startsWith('localhost') || /^127\./.test(domain) ? 'http' : 'https';
		// <service>.<project>.<domain> for specific service; <project>.<domain> for primary
		return `${scheme}://${serviceName}.${projectName}.${domain}/`;
	}
	return `/app/${encodeURIComponent(projectName)}/${encodeURIComponent(serviceName)}/`;
}
