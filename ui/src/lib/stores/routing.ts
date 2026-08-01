// Shared routing configuration store.
// Reads routing_mode ('path' | 'subdomain') and base_domain from /api/settings
// and provides helpers to generate absolute project/service URLs for both modes.
import { writable, get } from 'svelte/store';

export interface RoutingConfig {
	mode: 'path' | 'subdomain';
	baseDomain: string; // e.g. "klouds.online", "140.245.1.2", "localhost:8090"
}

export const routingConfig = writable<RoutingConfig>({
	mode: 'path',
	baseDomain: 'localhost:8090'
});

export async function loadRoutingConfig(): Promise<void> {
	try {
		const res = await fetch('/api/settings');
		if (res.ok) {
			const s = await res.json();
			routingConfig.set({
				mode: s.routing_mode === 'subdomain' ? 'subdomain' : 'path',
				baseDomain: s.base_domain || 'localhost:8090'
			});
		}
	} catch (e) {
		console.error('Failed to load routing config:', e);
	}
}

/** Returns true for local/private domains that should use http:// */
function isLocalDomain(domain: string): boolean {
	const host = domain.split(':')[0]; // strip port
	return (
		host === 'localhost' ||
		/^127\./.test(host) ||
		/^192\.168\./.test(host) ||
		/^10\./.test(host) ||
		/^172\.(1[6-9]|2\d|3[01])\./.test(host)
	);
}

/** Derive scheme from domain — http for local, https for public domains. */
export function schemeFor(domain: string): 'http' | 'https' {
	return isLocalDomain(domain) ? 'http' : 'https';
}

/**
 * Returns the absolute public URL for a project's primary service.
 *  - Subdomain mode: https://<project>.<baseDomain>/
 *  - Path mode:      https://<baseDomain>/app/<project>/
 */
export function getProjectUrl(projectName: string, config?: RoutingConfig): string {
	const cfg = config ?? get(routingConfig);
	const domain = cfg.baseDomain || 'localhost:8090';
	const scheme = schemeFor(domain);

	if (cfg.mode === 'subdomain') {
		return `${scheme}://${projectName}.${domain}/`;
	}
	// Path mode — always generate an absolute URL so it works from any browser
	return `${scheme}://${domain}/app/${encodeURIComponent(projectName)}/`;
}

/**
 * Returns the absolute public URL for a specific service within a project.
 *  - Subdomain mode: https://<service>.<project>.<baseDomain>/
 *  - Path mode:      https://<baseDomain>/app/<project>/<service>/
 */
export function getServiceUrl(
	projectName: string,
	serviceName: string,
	config?: RoutingConfig
): string {
	const cfg = config ?? get(routingConfig);
	const domain = cfg.baseDomain || 'localhost:8090';
	const scheme = schemeFor(domain);

	if (cfg.mode === 'subdomain') {
		return `${scheme}://${serviceName}.${projectName}.${domain}/`;
	}
	return `${scheme}://${domain}/app/${encodeURIComponent(projectName)}/${encodeURIComponent(serviceName)}/`;
}
