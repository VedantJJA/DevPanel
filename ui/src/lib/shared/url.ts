export function buildProjectUrl(opts: {
	routingMode?: string;
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

	return `https://${safeName}.${cleanDomain}${cleanPath}`;
}
