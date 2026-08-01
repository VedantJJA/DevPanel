import { describe, it, expect } from 'vitest';
import { buildProjectUrl } from './url';

describe('buildProjectUrl', () => {
	it('should format URL for path mode', () => {
		const url = buildProjectUrl({
			routingMode: 'path',
			rootDomain: 'example.com',
			projectName: 'my-app'
		});
		expect(url).toBe('https://example.com/app/my-app');
	});

	it('should format URL for subdomain mode', () => {
		const url = buildProjectUrl({
			routingMode: 'subdomain',
			rootDomain: 'example.com',
			projectName: 'my-app'
		});
		expect(url).toBe('https://my-app.example.com');
	});

	it('should URL-encode project names with special characters', () => {
		const url = buildProjectUrl({
			routingMode: 'path',
			rootDomain: 'example.com',
			projectName: 'my app/test & 123'
		});
		expect(url).toBe('https://example.com/app/my%20app%2Ftest%20%26%20123');
	});

	it('should append optional sub-path correctly in both modes', () => {
		const pathUrl = buildProjectUrl({
			routingMode: 'path',
			rootDomain: 'example.com',
			projectName: 'my-app',
			path: '/api/v1/health'
		});
		expect(pathUrl).toBe('https://example.com/app/my-app/api/v1/health');

		const subUrl = buildProjectUrl({
			routingMode: 'subdomain',
			rootDomain: 'example.com',
			projectName: 'my-app',
			path: 'dashboard'
		});
		expect(subUrl).toBe('https://my-app.example.com/dashboard');
	});
});
