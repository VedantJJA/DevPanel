export function buildProjectUrl(opts: {
	routingMode: 'path' | 'subdomain';
	rootDomain: string;
	projectName: string;
	path?: string;
}): string {
	const safeName = encodeURIComponent(opts.projectName);
	const path = opts.path ?? '';
	const cleanPath = path.startsWith('/') ? path : `/${path}`;
	const cleanDomain = opts.rootDomain
		.replace(/^panel\./, '')
		.replace(/^\./, '')
		.replace(/\/$/, '');
	if (opts.routingMode === 'subdomain') {
		return `https://${safeName}.${cleanDomain}${cleanPath}`;
	}
	return `https://${cleanDomain}/app/${safeName}${cleanPath}`;
}
