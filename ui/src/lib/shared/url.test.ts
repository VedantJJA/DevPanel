import { buildProjectUrl } from './url';

function assertEqual(actual: string, expected: string, label: string): void {
	if (actual !== expected) {
		throw new Error(`[${label}] Expected ${expected}, got ${actual}`);
	}
}

export function testBuildProjectUrl(): void {
	assertEqual(
		buildProjectUrl({
			routingMode: 'path',
			rootDomain: 'example.com',
			projectName: 'my-project'
		}),
		'https://example.com/app/my-project/',
		'path mode'
	);

	assertEqual(
		buildProjectUrl({
			routingMode: 'path',
			rootDomain: 'example.com',
			projectName: 'my-project',
			path: '/dashboard'
		}),
		'https://example.com/app/my-project/dashboard',
		'path mode with sub-path'
	);

	assertEqual(
		buildProjectUrl({
			routingMode: 'subdomain',
			rootDomain: 'example.com',
			projectName: 'my-project'
		}),
		'https://my-project.example.com/',
		'subdomain mode'
	);

	assertEqual(
		buildProjectUrl({
			routingMode: 'subdomain',
			rootDomain: 'example.com',
			projectName: 'my-project',
			path: '/api/v1'
		}),
		'https://my-project.example.com/api/v1',
		'subdomain mode with sub-path'
	);

	assertEqual(
		buildProjectUrl({
			routingMode: 'path',
			rootDomain: 'example.com',
			projectName: 'demo project'
		}),
		'https://example.com/app/demo%20project/',
		'encoding project name'
	);
}

if (typeof process !== 'undefined' && process.env.NODE_ENV === 'test') {
	testBuildProjectUrl();
}
