// Shared routing configuration store.
import { writable, get } from 'svelte/store';
import { loadConfig, getConfig } from '$lib/runtime/config';
import { buildProjectUrl as buildUrl } from '$lib/shared/url';

export interface RoutingConfig {
	mode: 'path' | 'subdomain';
	baseDomain: string; // e.g. "example.com", "140.245.1.2:8090", "localhost:8090"
}

export const routingConfig = writable<RoutingConfig>({
	mode: 'path',
	baseDomain: typeof window !== 'undefined' ? window.location.host.replace(/^panel\./, '') : 'localhost:8090'
});

export async function loadRoutingConfig(): Promise<void> {
	try {
		const cfg = await loadConfig();
		routingConfig.set({
			mode: cfg.routingMode,
			baseDomain: cfg.rootDomain
		});
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

/** Derive scheme from domain or browser context — http for local, https for public domains. */
export function schemeFor(domain: string): 'http' | 'https' {
	if (typeof window !== 'undefined' && window.location.protocol) {
		return window.location.protocol.replace(':', '') as 'http' | 'https';
	}
	return isLocalDomain(domain) ? 'http' : 'https';
}

/**
 * Returns the absolute public URL for a project's primary service.
 */
export function getProjectUrl(projectName: string, config?: RoutingConfig): string {
	const cfg = config ?? get(routingConfig);
	const runtimeCfg = getConfig();
	const mode = cfg?.mode || runtimeCfg.routingMode;
	const rootDomain = cfg?.baseDomain || runtimeCfg.rootDomain;
	return buildUrl({
		routingMode: mode,
		rootDomain: rootDomain,
		projectName: projectName,
		path: '/'
	});
}

/**
 * Returns the absolute public URL for a specific service within a project.
 * Uses the service's unique slug (Render-style) for the URL path/subdomain.
 */
export function getServiceUrl(
	projectName: string,
	serviceName: string,
	config?: RoutingConfig,
	serviceSlug?: string
): string {
	const cfg = config ?? get(routingConfig);
	const runtimeCfg = getConfig();
	const mode = cfg?.mode || runtimeCfg.routingMode;
	const rootDomain = cfg?.baseDomain || runtimeCfg.rootDomain;

	// Prefer slug over service name for URL generation (Render-style)
	const urlIdentifier = serviceSlug || serviceName || projectName;

	if (mode === 'subdomain') {
		return buildUrl({
			routingMode: 'subdomain',
			rootDomain: rootDomain,
			projectName: urlIdentifier,
			path: '/'
		});
	}
	return buildUrl({
		routingMode: 'path',
		rootDomain: rootDomain,
		projectName: urlIdentifier,
		path: '/'
	});
}
