import { get } from 'svelte/store';
import { routingConfig, schemeFor } from './stores/routing';
import type { ScanResult, ProjectDetail, ServiceRecord, DeploymentRecord } from './types';

/**
 * Resolves the API endpoint URL dynamically:
 * Checks VITE_BACKEND_URL or VITE_API_URL environment configuration if present,
 * or resolves based on the current host.
 */
export function getApiUrl(path: string): string {
	const cleanPath = path.startsWith('/') ? path : '/' + path;
	const envBackend = (import.meta as any).env?.VITE_BACKEND_URL || (import.meta as any).env?.VITE_API_URL;
	if (envBackend) {
		return `${envBackend.replace(/\/$/, '')}${cleanPath}`;
	}
	return cleanPath;
}

/**
 * Resolves the dynamic backend API base URL for a hosted project service.
 * Uses the service's unique slug (Render-style) for routing.
 * - Path mode: https://<domain>/app/<slug>/
 * - Subdomain mode: https://<slug>.<domain>/
 *
 * When calling a backend API from a frontend service, pass the backend service's slug
 * so requests route to the correct container.
 */
export function getProjectServiceApiUrl(
	projectName: string,
	serviceName?: string,
	path: string = '',
	serviceSlug?: string
): string {
	const cfg = get(routingConfig);
	const domain =
		cfg.baseDomain || (typeof window !== 'undefined' ? window.location.host.replace(/^panel\./, '') : 'localhost:8090');
	const scheme = schemeFor(domain);
	const cleanPath = path ? (path.startsWith('/') ? path : '/' + path) : '';

	// Use slug for URL construction (Render-style)
	const urlIdentifier = serviceSlug || serviceName || projectName;

	if (cfg.mode === 'subdomain') {
		const hostWithoutPort = domain.split(':')[0];
		const portSuffix = domain.includes(':') ? ':' + domain.split(':')[1] : '';
		return `${scheme}://${urlIdentifier}.${hostWithoutPort}${portSuffix}${cleanPath}`;
	}

	return `${scheme}://${domain}/app/${encodeURIComponent(urlIdentifier)}${cleanPath}`;
}

export async function scanRepo(repoUrl: string, appName: string): Promise<ScanResult> {
	const res = await fetch(getApiUrl('/api/repos/scan'), {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ repo_url: repoUrl, app_name: appName })
	});
	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: 'Scan request failed' }));
		throw new Error(err.error || 'Failed to scan repository');
	}
	return res.json();
}

export async function createProject(payload: {
	app_name: string;
	repo_url: string;
	blueprint: any;
	services: any[];
}): Promise<ProjectDetail> {
	const res = await fetch(getApiUrl('/api/projects'), {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(payload)
	});
	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: 'Project creation failed' }));
		throw new Error(err.error || 'Failed to create project');
	}
	return res.json();
}

export async function listProjects(): Promise<{ projects: any[] }> {
	const res = await fetch(getApiUrl('/api/projects'));
	if (!res.ok) throw new Error('Failed to list projects');
	return res.json();
}

export async function getProject(id: string): Promise<ProjectDetail> {
	const res = await fetch(getApiUrl(`/api/projects/${encodeURIComponent(id)}`));
	if (!res.ok) throw new Error('Failed to fetch project details');
	return res.json();
}

export async function updateService(
	projectId: string,
	serviceName: string,
	updates: Partial<ServiceRecord>
): Promise<ServiceRecord> {
	const res = await fetch(
		getApiUrl(`/api/projects/${encodeURIComponent(projectId)}/services/${encodeURIComponent(serviceName)}`),
		{
			method: 'PATCH',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(updates)
		}
	);
	if (!res.ok) throw new Error('Failed to update service config');
	return res.json();
}

export async function triggerDeploy(
	projectId: string
): Promise<{ deployment_id: string; project_id: string; status: string; log_url: string }> {
	const res = await fetch(getApiUrl(`/api/projects/${encodeURIComponent(projectId)}/deploy`), {
		method: 'POST'
	});
	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: 'Deployment trigger failed' }));
		throw new Error(err.error || 'Failed to trigger deployment');
	}
	return res.json();
}

export async function listDeployments(projectId: string): Promise<{ deployments: DeploymentRecord[] }> {
	const res = await fetch(getApiUrl(`/api/projects/${encodeURIComponent(projectId)}/deployments`));
	if (!res.ok) throw new Error('Failed to fetch deployment history');
	return res.json();
}

export async function restartService(projectId: string, serviceName: string): Promise<any> {
	const res = await fetch(
		getApiUrl(`/api/projects/${encodeURIComponent(projectId)}/services/${encodeURIComponent(serviceName)}/restart`),
		{
			method: 'POST'
		}
	);
	if (!res.ok) throw new Error('Failed to restart service container');
	return res.json();
}
