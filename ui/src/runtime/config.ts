export interface RuntimeConfig {
	rootDomain: string;
	routingMode: 'path' | 'subdomain';
	apiBase: string;
}

let currentConfig: RuntimeConfig = {
	rootDomain: typeof window !== 'undefined' ? window.location.host.replace(/^panel\./, '') : 'localhost:8090',
	routingMode: 'path',
	apiBase: '/api'
};

export async function fetchConfig(): Promise<RuntimeConfig> {
	try {
		const res = await fetch('/api/config');
		if (res.ok) {
			const data = await res.json();
			currentConfig = {
				rootDomain: data.rootDomain || currentConfig.rootDomain,
				routingMode: data.routingMode === 'subdomain' ? 'subdomain' : 'path',
				apiBase: data.apiBase || '/api'
			};
		}
	} catch (e) {
		console.error('Failed to fetch runtime config from /api/config:', e);
	}
	return currentConfig;
}

export function getConfig(): RuntimeConfig {
	return currentConfig;
}
