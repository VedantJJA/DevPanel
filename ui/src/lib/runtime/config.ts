export interface RuntimeConfig {
	rootDomain: string;
	routingMode: 'path' | 'subdomain';
	apiBase: string;
}

let cachedConfig: RuntimeConfig = {
	rootDomain: typeof window !== 'undefined' ? window.location.host.replace(/^panel\./, '') : 'localhost:8090',
	routingMode: 'path',
	apiBase: '/api'
};

export async function loadConfig(): Promise<RuntimeConfig> {
	try {
		const res = await fetch('/api/config');
		if (res.ok) {
			const data = await res.json();
			cachedConfig = {
				rootDomain: data.rootDomain || cachedConfig.rootDomain,
				routingMode: data.routingMode === 'subdomain' ? 'subdomain' : 'path',
				apiBase: data.apiBase || '/api'
			};
		}
	} catch (e) {
		console.error('Failed to load runtime config from /api/config:', e);
	}
	return cachedConfig;
}

export function getConfig(): RuntimeConfig {
	return cachedConfig;
}
