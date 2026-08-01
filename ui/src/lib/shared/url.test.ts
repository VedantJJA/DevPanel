import { describe, it } from 'node:test';
import assert from 'node:assert';
import { buildProjectUrl } from './url.ts';

describe('buildProjectUrl', () => {
	it('builds URL for path mode without sub-path', () => {
		const url = buildProjectUrl({
			routingMode: 'path',
			rootDomain: 'example.com',
			projectName: 'my-project'
		});
		assert.strictEqual(url, 'https://example.com/app/my-project/');
	});

	it('builds URL for path mode with optional sub-path', () => {
		const url = buildProjectUrl({
			routingMode: 'path',
			rootDomain: 'example.com',
			projectName: 'my-project',
			path: '/dashboard'
		});
		assert.strictEqual(url, 'https://example.com/app/my-project/dashboard');
	});

	it('builds URL for subdomain mode without sub-path', () => {
		const url = buildProjectUrl({
			routingMode: 'subdomain',
			rootDomain: 'example.com',
			projectName: 'my-project'
		});
		assert.strictEqual(url, 'https://my-project.example.com/');
	});

	it('builds URL for subdomain mode with optional sub-path', () => {
		const url = buildProjectUrl({
			routingMode: 'subdomain',
			rootDomain: 'example.com',
			projectName: 'my-project',
			path: '/api/v1'
		});
		assert.strictEqual(url, 'https://my-project.example.com/api/v1');
	});

	it('URL-encodes special characters in project name', () => {
		const url = buildProjectUrl({
			routingMode: 'path',
			rootDomain: 'example.com',
			projectName: 'demo project'
		});
		assert.strictEqual(url, 'https://example.com/app/demo%20project/');
	});
});
