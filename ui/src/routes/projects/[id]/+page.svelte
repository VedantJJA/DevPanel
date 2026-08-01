<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import AppShell, { type ProjectContext } from '$lib/components/AppShell.svelte';
	import EnvVarEditor from '$lib/components/EnvVarEditor.svelte';
	import Terminal from '$lib/components/Terminal.svelte';
	import { getProject, updateService, triggerDeploy, restartService } from '$lib/api';
	import type { ProjectDetail, ServiceRecord, SystemStats } from '$lib/types';
	import { createLogStore } from '$lib/stores/logs';
	import { routingConfig, loadRoutingConfig, getProjectUrl, getServiceUrl } from '$lib/stores/routing';


	let projectId = $derived(page.params.id || '');

	// --- Build log store (WebSocket-backed, auto-clears on new deploy) ---
	let buildLogStore = $derived(createLogStore(projectId));
	let logScrollEl = $state<HTMLElement | null>(null);

	// Auto-scroll log container to bottom on new entries
	$effect(() => {
		if (logScrollEl && buildLogStore) {
			// Reading $buildLogStore triggers reactive update
			logScrollEl.scrollTop = logScrollEl.scrollHeight;
		}
	});

	// Trigger a new project deploy via the existing SSE pipeline
	async function startDeploy() {
		if (!projectId) return;
		await triggerDeploy(projectId);
	}

	// Clear build logs both server-side and locally
	function clearBuildLogs() {
		buildLogStore.clear();
	}

	let projectData = $state<ProjectDetail | null>(null);
	let activeServiceIdx = $state(0);
	let activeTab = $state<string>('logs');
	let logType = $state<'build' | 'live' | '1h' | '24h'>('build');

	let isSaving = $state(false);
	let saveMessage = $state<string | null>(null);
	let systemStats = $state<SystemStats>({ totalContainers: 0, activeContainers: 0, stoppedContainers: 0, totalMemMb: 0, usedMemMb: 0, memPercent: 0, cpus: 1 });
	let pingMs = $state<number | null>(null);

	// Settings state for additional fields
	let selectedInstanceType = $state<'shared' | 'standard' | 'performance'>('shared');
	let autoDeployEnabled = $state(true);
	let preDeployCommand = $state('npm run migrate');
	let maintenanceMode = $state(false);
	let replicaCount = $state(1);

	// Container Telemetry State for active service container
	let serviceMetrics = $state({
		cpuPercent: 0,
		memoryMb: 0,
		status: 'Healthy'
	});
	let metricsInterval: any = null;

	let currentSvc = $derived(projectData?.services?.[activeServiceIdx] || projectData?.services?.[0]);

	// Stream logs directly for the selected individual service container
	let sseUrl = $derived.by(() => {
		if (!currentSvc) return `/api/projects/${projectId}/logs`;
		if (logType === 'build') {
			return `/api/projects/${projectId}/logs?service=${encodeURIComponent(currentSvc.name)}`;
		}
		if (logType === 'live') {
			return `/api/projects/${projectId}/services/${encodeURIComponent(currentSvc.name)}/logs`;
		}
		return `/api/projects/${projectId}/services/${encodeURIComponent(currentSvc.name)}/logs?since=${logType}`;
	});

	// Dynamic Audit Events from real project deployment history
	let auditEvents = $state<any[]>([]);

	// Service-Specific Side Navigation Groups matching Stitch CloudStack templates
	let projectContext = $derived<ProjectContext>({
		projectName: projectData?.blueprint?.name || projectId,
		services: projectData?.services || [],
		activeServiceIdx,
		onSelectService: (idx: number) => { activeServiceIdx = idx; activeTab = 'events'; fetchServiceMetrics(); },
		activeTab,
		onSelectTab: (tabId: string) => { activeTab = tabId; },
		groups: currentSvc?.type === 'postgres' || currentSvc?.type === 'database'
			? [
					{
						title: 'DASHBOARD',
						items: [
							{ id: 'events', label: 'General Info', icon: 'database' },
							{ id: 'connections', label: 'Connections', icon: 'link' },
							{ id: 'settings', label: 'DB Settings', icon: 'settings' }
						]
					},
					{
						title: 'MONITOR',
						items: [
							{ id: 'metrics', label: 'Metrics', icon: 'query_stats' }
						]
					},
					{
						title: 'MANAGE',
						items: [
							{ id: 'environment', label: 'Credentials & Passwords', icon: 'key' },
							{ id: 'volumes', label: 'Storage & Volume', icon: 'storage' }
						]
					}
			  ]
			: currentSvc?.type === 'static'
			? [
					{
						title: 'DASHBOARD',
						items: [
							{ id: 'logs', label: 'Build & Deploy Logs', icon: 'terminal' },
							{ id: 'events', label: 'Events Audit', icon: 'event_note' },
							{ id: 'settings', label: 'Build & Deploy', icon: 'settings' }
						]
					},
					{
						title: 'MONITOR',
						items: [
							{ id: 'metrics', label: 'Edge Metrics', icon: 'query_stats' }
						]
					},
					{
						title: 'MANAGE',
						items: [
							{ id: 'environment', label: 'Environment', icon: 'key' },
							{ id: 'domains', label: 'Domains & SSL', icon: 'globe' },
							{ id: 'previews', label: 'PR Previews', icon: 'visibility' },
							{ id: 'redirects', label: 'Redirects & Rewrites', icon: 'router' },
							{ id: 'headers', label: 'Headers', icon: 'view_headline' }
						]
					}
			  ]
			: [
					{
						title: 'DASHBOARD',
						items: [
							{ id: 'logs', label: 'Build & Live Logs', icon: 'terminal' },
							{ id: 'events', label: 'Events Audit', icon: 'event_note' },
							{ id: 'settings', label: 'Service Settings', icon: 'settings' }
						]
					},
					{
						title: 'MONITOR',
						items: [
							{ id: 'metrics', label: 'Metrics', icon: 'query_stats' }
						]
					},
					{
						title: 'MANAGE',
						items: [
							{ id: 'environment', label: 'Environment', icon: 'key' },
							{ id: 'shell', label: 'Container Shell', icon: 'terminal' },
							{ id: 'scaling', label: 'Scaling & Compute', icon: 'dynamic_form' },
							{ id: 'previews', label: 'PR Previews', icon: 'visibility' },
							{ id: 'domains', label: 'Domains & SSL', icon: 'globe' },
							{ id: 'volumes', label: 'Disk Storage', icon: 'storage' },
							{ id: 'jobs', label: 'One-Off Jobs', icon: 'bolt' }
						]
					}
			  ]
	});

	function formatTimeAgo(isoString: string): string {
		if (!isoString) return 'Just now';
		const date = new Date(isoString);
		const now = new Date();
		const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);
		if (seconds < 10) return 'Just now';
		if (seconds < 60) return `${seconds}s ago`;
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes} min${minutes > 1 ? 's' : ''} ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours} hour${hours > 1 ? 's' : ''} ago`;
		const days = Math.floor(hours / 24);
		return `${days} day${days > 1 ? 's' : ''} ago`;
	}

	async function loadProjectData() {
		if (!projectId) return;
		try {
			const [res, depRes] = await Promise.all([
				getProject(projectId),
				fetch(`/api/projects/${projectId}/deployments`).then(r => r.ok ? r.json() : { deployments: [] }).catch(() => ({ deployments: [] }))
			]);
			projectData = res;

			if (res?.blueprint?.status === 'building' || res?.blueprint?.status === 'deploying' || res?.latest?.status === 'building') {
				activeTab = 'logs';
				logType = 'build';
			}

			// Auto-select primary web/static service if initial index points to database
			if (res.services && res.services.length > 0) {
				const webIdx = res.services.findIndex((s: any) => s.type === 'web' || s.type === 'static');
				if (webIdx >= 0 && activeServiceIdx === 0) {
					activeServiceIdx = webIdx;
				}
			}

			const deps = depRes.deployments || [];
			if (deps.length > 0) {
				auditEvents = deps.map((d: any, idx: number) => ({
					id: d.id || String(idx),
					type: 'Deployment',
					title: d.status === 'live' || d.status === 'active' || d.status === 'success'
						? `Deployment complete — ${d.id}`
						: d.status === 'building' || d.status === 'deploying'
						? `Redeployment in progress`
						: `Deployment failed — ${d.error_msg || 'Build error'}`,
					time: d.created_at ? formatTimeAgo(d.created_at) : 'Just now',
					status: d.status === 'live' || d.status === 'active' || d.status === 'success' ? 'Success' : d.status === 'building' ? 'Building' : 'Failed',
					icon: d.status === 'live' || d.status === 'active' ? 'rocket_launch' : d.status === 'building' ? 'sync' : 'error',
					user: d.trigger === 'manual' ? 'VedantJJA' : 'System',
					color: d.status === 'live' || d.status === 'active' || d.status === 'success' ? 'var(--success)' : d.status === 'building' ? 'var(--primary)' : 'var(--error)'
				}));
			} else {
				auditEvents = [
					{ id: '1', type: 'Deployment', title: `Initial service setup for ${res.blueprint?.name || projectId}`, time: 'Just now', status: 'Ready', icon: 'rocket_launch', user: 'System', color: 'var(--primary)' }
				];
			}
		} catch (e) {
			console.error('Failed to load project details:', e);
		}
	}

	async function measurePing() {
		const start = performance.now();
		try {
			const r = await fetch('/healthz', { cache: 'no-store' });
			if (r.ok) pingMs = Math.round(performance.now() - start);
		} catch { pingMs = null; }
	}

	async function handleSaveService(svc: ServiceRecord) {
		isSaving = true;
		saveMessage = null;
		try {
			await updateService(projectId, svc.name, {
				env_vars: svc.env_vars,
				port: svc.port,
				custom_domain: svc.custom_domain,
				build_command: svc.build_command,
				start_command: svc.start_command,
				runtime: svc.runtime
			});
			saveMessage = `Saved settings for ${svc.name}! Triggering redeployment...`;
			await triggerDeploy(projectId);
			await loadProjectData();
		} catch (e: any) {
			saveMessage = `Failed to save settings: ${e.message}`;
		} finally {
			isSaving = false;
		}
	}

	async function handleRestart(svcName: string) {
		try {
			await restartService(projectId, svcName);
			saveMessage = `Container for ${svcName} restarted successfully.`;
		} catch (e: any) {
			saveMessage = `Restart failed: ${e.message}`;
		}
	}

	async function fetchServiceMetrics() {
		try {
			const [cRes, sRes] = await Promise.all([fetch('/api/containers'), fetch('/api/system/stats')]);
			if (sRes.ok) { const s = await sRes.json(); systemStats = { ...systemStats, ...s }; }
			if (cRes.ok && projectData?.services[activeServiceIdx]) {
				const data = await cRes.json();
				const list = data.containers || [];
				const targetSvc = projectData.services[activeServiceIdx];
				const targetName = `devpnl-${projectId.replace(/^bp-/, '')}-${targetSvc.name}`;
				const container = list.find((c: any) => c.name === targetName || c.name.includes(targetSvc.name));
				if (container) {
					serviceMetrics = {
						cpuPercent: container.cpuPercent || 0,
						memoryMb: container.memoryMb || 0,
						status: container.status === 'running' ? 'Healthy' : 'Stopped'
					};
				}
			}
		} catch (e) { console.error('Metrics error:', e); }
	}

	function copyToClipboard(text: string) {
		if (navigator.clipboard) {
			navigator.clipboard.writeText(text);
			saveMessage = `Copied to clipboard: "${text.slice(0, 35)}..."`;
			setTimeout(() => { if (saveMessage?.startsWith('Copied')) saveMessage = null; }, 3000);
		}
	}

	onMount(() => {
		loadRoutingConfig();
		const urlTab = page.url.searchParams.get('tab');

		if (urlTab) {
			activeTab = urlTab;
			if (urlTab === 'logs') {
				logType = 'build';
			}
		} else if (page.url.searchParams.get('deploying') === 'true') {
			activeTab = 'logs';
			logType = 'build';
		}
		loadProjectData();
		measurePing();
		metricsInterval = setInterval(fetchServiceMetrics, 3000);
	});

	onDestroy(() => {
		if (metricsInterval) clearInterval(metricsInterval);
	});
</script>

<AppShell {systemStats} {pingMs} {projectContext}>
	<!-- Top Breadcrumbs Header -->
	<div class="mb-6">
		<div class="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-wider" style="color: var(--on-surface-variant)">
			<a href="/blueprints" class="hover:underline" style="color: var(--primary)">Projects</a>
			<span class="material-symbols-outlined text-sm">chevron_right</span>
			<span class="font-bold" style="color: var(--on-surface)">{projectData?.blueprint?.name || projectId}</span>
			{#if currentSvc && currentSvc.name !== (projectData?.blueprint?.name || projectId)}
				<span class="material-symbols-outlined text-sm">chevron_right</span>
				<span class="font-bold" style="color: var(--primary)">{currentSvc.name}</span>
			{/if}
		</div>

		{#if saveMessage}
			<div class="mb-4 flex items-center justify-between rounded-xl border px-4 py-3 text-sm shadow-sm" style={saveMessage.includes('Failed') ? 'background-color: var(--error-container); border-color: var(--error); color: var(--error)' : 'background-color: var(--success-container); border-color: var(--success); color: var(--on-success-container)'}>
				<span>{saveMessage}</span>
				<button onclick={() => (saveMessage = null)} class="font-bold">✕</button>
			</div>
		{/if}
	</div>

	{#if projectData && currentSvc}

		<!-- ========================================================================= -->
		<!-- SERVICE VIEW 1: POSTGRESQL / DATABASE (postgresql_info_cloudstack)       -->
		<!-- ========================================================================= -->
		{#if currentSvc.type === 'postgres' || currentSvc.type === 'database'}
			<div class="space-y-6">
				<!-- Header section -->
				<div class="flex flex-wrap items-end justify-between gap-4">
					<div>
						<div class="flex items-center gap-3">
							<span class="flex h-12 w-12 items-center justify-center rounded-xl shadow-sm" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 28px">database</span>
							</span>
							<div>
								<div class="flex items-center gap-2">
									<h1 class="text-2xl font-bold lg:text-3xl" style="color: var(--on-surface)">{currentSvc.name}</h1>
									<span class="rounded-full px-3 py-0.5 text-xs font-bold uppercase" style="background-color: var(--success-container); color: var(--on-success-container)">
										{serviceMetrics.status}
									</span>
								</div>
								<p class="mt-0.5 text-xs" style="color: var(--on-surface-variant)">Managed PostgreSQL Container · Engine 15-alpine</p>
							</div>
						</div>
					</div>

					<div class="flex flex-wrap gap-2">
						<button onclick={() => handleRestart(currentSvc!.name)} class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors hover:opacity-80" style="border-color: var(--outline-variant); background-color: var(--surface-lowest);">
							<span class="material-symbols-outlined" style="font-size: 18px">restart_alt</span>Restart DB
						</button>
						<button onclick={() => copyToClipboard(`postgres://postgres:${currentSvc!.env_vars?.['POSTGRES_PASSWORD'] || 'postgres'}@localhost:${currentSvc!.port || 5432}/${currentSvc!.env_vars?.['POSTGRES_DB'] || currentSvc!.name}`)} class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-opacity hover:opacity-90" style="background-color: var(--primary); color: var(--on-primary);">
							<span class="material-symbols-outlined" style="font-size: 18px">content_copy</span>Copy Connection URI
						</button>
					</div>
				</div>

				<!-- TAB CONTENT DEPENDING ON SIDEBAR CLICK -->
				{#if activeTab === 'events' || activeTab === 'overview'}
					<!-- PostgreSQL Grid Layout from postgresql_info_cloudstack -->
					<div class="grid grid-cols-12 gap-6">
						<!-- General Information Box -->
						<div class="col-span-12 lg:col-span-7 space-y-6">
							<div class="card-surface overflow-hidden">
								<div class="border-b px-6 py-4 flex items-center justify-between" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
									<h2 class="font-bold text-base flex items-center gap-2">
										<span class="material-symbols-outlined text-primary">database</span>
										General Information
									</h2>
									<span class="px-2.5 py-0.5 rounded-full text-xs font-bold" style="background-color: var(--success-container); color: var(--on-success-container)">Active</span>
								</div>
								<div class="p-6 grid grid-cols-2 gap-y-6 gap-x-8 text-sm">
									<div>
										<span class="label-caps block mb-1" style="color: var(--on-surface-variant)">Instance Name</span>
										<p class="font-semibold">{currentSvc.name}</p>
									</div>
									<div>
										<span class="label-caps block mb-1" style="color: var(--on-surface-variant)">PG Version</span>
										<p class="font-semibold">15.4 (Alpine)</p>
									</div>
									<div>
										<span class="label-caps block mb-1" style="color: var(--on-surface-variant)">Allocated Port</span>
										<p class="font-semibold font-mono">{currentSvc.port || 5432}</p>
									</div>
									<div>
										<span class="label-caps block mb-1" style="color: var(--on-surface-variant)">Database Name</span>
										<p class="font-semibold font-mono">{currentSvc.env_vars?.['POSTGRES_DB'] || currentSvc.name}</p>
									</div>
									<div>
										<span class="label-caps block mb-1" style="color: var(--on-surface-variant)">Region</span>
										<p class="font-semibold">us-east-1 (N. Virginia)</p>
									</div>
									<div>
										<span class="label-caps block mb-1" style="color: var(--on-surface-variant)">Instance Type</span>
										<p class="font-semibold">db.m6g.xlarge <span class="text-xs font-normal" style="color: var(--on-surface-variant)">(4 vCPU, 16GB RAM)</span></p>
									</div>
									<div class="col-span-2 pt-3 border-t flex items-center justify-between text-xs" style="border-color: var(--outline-variant)">
										<span class="flex items-center gap-2 font-semibold" style="color: var(--success)">
											<span class="material-symbols-outlined" style="font-size: 18px">verified_user</span> High Availability Enabled
										</span>
										<span style="color: var(--on-surface-variant)">Isolated docker container volume mount</span>
									</div>
								</div>
							</div>
						</div>

						<!-- Resources & Telemetry Right Column -->
						<div class="col-span-12 lg:col-span-5 space-y-6">
							<div class="card-surface p-6 space-y-4">
								<h2 class="font-bold text-base">Resource Metrics</h2>
								<div>
									<div class="flex justify-between text-xs mb-1">
										<span class="label-caps" style="color: var(--on-surface-variant)">CPU Utilisation</span>
										<span class="font-bold">{serviceMetrics.cpuPercent.toFixed(1)}%</span>
									</div>
									<div class="h-2 w-full rounded-full" style="background-color: var(--surface-high)">
										<div class="h-2 rounded-full transition-all" style="width: {serviceMetrics.cpuPercent}%; background-color: var(--primary)"></div>
									</div>
								</div>

								<div>
									<div class="flex justify-between text-xs mb-1">
										<span class="label-caps" style="color: var(--on-surface-variant)">Memory Usage</span>
										<span class="font-bold">{Math.round(serviceMetrics.memoryMb)} MB</span>
									</div>
									<div class="h-2 w-full rounded-full" style="background-color: var(--surface-high)">
										<div class="h-2 rounded-full transition-all" style="width: {Math.min((serviceMetrics.memoryMb/512)*100, 100)}%; background-color: var(--primary)"></div>
									</div>
								</div>
							</div>
						</div>
					</div>
				{/if}

				{#if activeTab === 'connections'}
					<!-- Dedicated Connections Page View -->
					<div class="card-surface overflow-hidden">
						<div class="border-b px-6 py-4 flex items-center justify-between" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<h2 class="font-bold text-base flex items-center gap-2">
								<span class="material-symbols-outlined text-primary">link</span>
								Connections &amp; Database Credentials
							</h2>
						</div>
						<div class="p-6 space-y-6 text-xs font-mono">
							<div>
								<span class="label-caps block mb-1" style="color: var(--on-surface-variant)">Internal Container Host</span>
								<div class="flex items-center justify-between rounded border p-3" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
									<span>{`devpnl-${projectId.replace(/^bp-/, '')}-${currentSvc.name}`}</span>
									<button onclick={() => copyToClipboard(`devpnl-${projectId.replace(/^bp-/, '')}-${currentSvc.name}`)} class="text-primary hover:opacity-80">
										<span class="material-symbols-outlined" style="font-size: 18px">content_copy</span>
									</button>
								</div>
							</div>

							<div>
								<span class="label-caps block mb-1" style="color: var(--on-surface-variant)">External Connection Port</span>
								<div class="flex items-center justify-between rounded border p-3" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
									<span>localhost:{currentSvc.port || 5432}</span>
									<button onclick={() => copyToClipboard(`localhost:${currentSvc!.port || 5432}`)} class="text-primary hover:opacity-80">
										<span class="material-symbols-outlined" style="font-size: 18px">content_copy</span>
									</button>
								</div>
							</div>

							<div>
								<span class="label-caps block mb-1" style="color: var(--on-surface-variant)">Full Postgres URI</span>
								<div class="flex items-center justify-between rounded border p-3 truncate" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
									<span class="truncate">{`postgres://postgres:${currentSvc.env_vars?.['POSTGRES_PASSWORD'] || 'postgres'}@localhost:${currentSvc.port || 5432}/${currentSvc.env_vars?.['POSTGRES_DB'] || currentSvc.name}`}</span>
									<button onclick={() => copyToClipboard(`postgres://postgres:${currentSvc!.env_vars?.['POSTGRES_PASSWORD'] || 'postgres'}@localhost:${currentSvc!.port || 5432}/${currentSvc!.env_vars?.['POSTGRES_DB'] || currentSvc!.name}`)} class="text-primary hover:opacity-80 ml-2">
										<span class="material-symbols-outlined" style="font-size: 18px">content_copy</span>
									</button>
								</div>
							</div>
						</div>
					</div>
				{/if}

				{#if activeTab === 'settings'}
					<div class="card-surface p-6 space-y-6">
						<h2 class="font-bold text-base">Database Configuration</h2>
						<div class="space-y-4 max-w-md">
							<div>
								<label for="pgPortInput" class="label-caps block text-xs mb-1">Database Port</label>
								<input id="pgPortInput" type="number" bind:value={currentSvc.port} class="w-full rounded border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
							</div>
							<button onclick={() => handleSaveService(currentSvc!)} class="flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 16px">save</span>Save &amp; Restart DB
							</button>
						</div>
					</div>
				{/if}

				{#if activeTab === 'environment'}
					<div class="card-surface p-6">
						<EnvVarEditor bind:envVars={currentSvc.env_vars} serviceName={currentSvc.name} />
						<div class="mt-4 flex justify-end">
							<button onclick={() => handleSaveService(currentSvc!)} class="flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 18px">save</span>Save &amp; Apply
							</button>
						</div>
					</div>
				{/if}

				{#if activeTab === 'logs'}
					<div class="h-[500px]">
						<Terminal projectId={projectId} serviceFilter={currentSvc.name} sourceUrl={sseUrl} title={`${currentSvc.name} — postgres logs`} />
					</div>
				{/if}

				{#if activeTab === 'volumes'}
					<div class="card-surface p-6 space-y-4">
						<h2 class="font-bold text-base">Mounted Storage Disks &amp; Volumes</h2>
						<div class="rounded border p-4 font-mono text-xs flex justify-between" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<span>/var/lib/postgresql/data ({currentSvc.name}_pgdata)</span>
							<span style="color: var(--success)">Mounted</span>
						</div>
					</div>
				{/if}

				{#if activeTab === 'metrics'}
					<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
						<div class="card-surface p-5">
							<span class="label-caps" style="color: var(--on-surface-variant)">CPU Utilisation</span>
							<div class="my-2 text-2xl font-bold">{serviceMetrics.cpuPercent.toFixed(1)}%</div>
							<div class="h-2 w-full rounded-full" style="background-color: var(--surface-high)">
								<div class="h-2 rounded-full transition-all" style="width: {serviceMetrics.cpuPercent}%; background-color: var(--primary)"></div>
							</div>
						</div>
						<div class="card-surface p-5">
							<span class="label-caps" style="color: var(--on-surface-variant)">Memory Usage</span>
							<div class="my-2 text-2xl font-bold">{Math.round(serviceMetrics.memoryMb)} MB</div>
							<div class="h-2 w-full rounded-full" style="background-color: var(--surface-high)">
								<div class="h-2 rounded-full transition-all" style="width: {Math.min((serviceMetrics.memoryMb/512)*100, 100)}%; background-color: var(--primary)"></div>
							</div>
						</div>
					</div>
				{/if}
			</div>

		<!-- ========================================================================= -->
		<!-- SERVICE VIEW 2: STATIC SITE (static_site_settings_cloudstack)             -->
		<!-- ========================================================================= -->
		{:else if currentSvc.type === 'static'}
			<div class="space-y-6">
				<!-- Header section -->
				<div class="flex flex-wrap items-end justify-between gap-4">
					<div>
						<div class="flex items-center gap-3">
							<span class="flex h-12 w-12 items-center justify-center rounded-xl shadow-sm" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 28px">globe</span>
							</span>
							<div>
								<div class="flex items-center gap-2">
									<h1 class="text-2xl font-bold lg:text-3xl" style="color: var(--on-surface)">{currentSvc.name}</h1>
									<span class="rounded-full px-3 py-0.5 text-xs font-bold uppercase" style="background-color: var(--success-container); color: var(--on-success-container)">
										Static Site
									</span>
								</div>
								<p class="mt-0.5 text-xs" style="color: var(--on-surface-variant)">Fast Edge Distribution · Auto-TLS Caddy Server</p>
							</div>
						</div>
					</div>

					<div class="flex flex-wrap gap-2">
						<a href={getProjectUrl(projectData?.blueprint?.name || projectId, $routingConfig)} target="_blank" rel="noreferrer" class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors hover:opacity-80" style="border-color: var(--outline-variant); background-color: var(--surface-lowest);">
							<span class="material-symbols-outlined" style="font-size: 18px">open_in_new</span>Visit Live Site
						</a>
						<button onclick={() => triggerDeploy(projectId)} class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-opacity hover:opacity-90" style="background-color: var(--primary); color: var(--on-primary);">
							<span class="material-symbols-outlined" style="font-size: 18px">rocket_launch</span>Redeploy Build
						</button>
					</div>
				</div>

				<!-- Dedicated Tab Views for Static Site -->
				{#if activeTab === 'events' || activeTab === 'overview'}
					<div class="card-surface p-6 space-y-6">
						<div class="flex items-center justify-between border-b pb-4" style="border-color: var(--outline-variant)">
							<h2 class="font-bold text-base">Service Audit Events</h2>
							<span class="label-caps text-xs" style="color: var(--on-surface-variant)">Showing recent deployments &amp; build history</span>
						</div>
						<div class="space-y-3">
							{#each auditEvents as evt}
								<div class="flex items-start gap-4 rounded-lg border p-4 transition-colors" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
									<span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg" style="background-color: var(--surface-low); color: {evt.color}">
										<span class="material-symbols-outlined" style="font-size: 20px">{evt.icon}</span>
									</span>
									<div class="min-w-0 flex-1">
										<div class="flex items-center justify-between">
											<h3 class="font-semibold text-sm">{evt.title}</h3>
											<span class="label-caps text-[10px]" style="color: var(--on-surface-variant)">{evt.time}</span>
										</div>
										<p class="mt-1 text-xs" style="color: var(--on-surface-variant)">Triggered by {evt.user} · Event #{evt.id}</p>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				{#if activeTab === 'settings'}
					<div class="card-surface p-6 space-y-6 max-w-3xl">
						<h2 class="font-bold text-base border-b pb-3" style="border-color: var(--outline-variant)">Build &amp; Deployment Configuration</h2>
						<form onsubmit={(e) => { e.preventDefault(); handleSaveService(currentSvc!); }} class="space-y-6">
							<div>
								<label for="stName" class="label-caps block text-xs mb-1">Project Name</label>
								<input id="stName" type="text" bind:value={currentSvc.name} class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
							</div>

							<div>
								<label for="stBuildCmd" class="label-caps block text-xs mb-1">Build Command</label>
								<input id="stBuildCmd" type="text" bind:value={currentSvc.build_command} placeholder="npm run build" class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
							</div>

							<div>
								<label for="stPublishDir" class="label-caps block text-xs mb-1">Publish Output Directory</label>
								<input id="stPublishDir" type="text" placeholder="dist or build" class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
							</div>

							<div class="flex justify-end pt-4 border-t" style="border-color: var(--outline-variant)">
								<button type="submit" disabled={isSaving} class="flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
									<span class="material-symbols-outlined" style="font-size: 18px">save</span>
									{isSaving ? 'Saving...' : 'Save & Trigger Build'}
								</button>
							</div>
						</form>
					</div>
				{/if}

				{#if activeTab === 'previews'}
					<div class="card-surface p-6 space-y-4">
						<h2 class="font-bold text-base flex items-center gap-2">
							<span class="material-symbols-outlined text-primary">visibility</span> PR Previews
						</h2>
						<p class="text-xs" style="color: var(--on-surface-variant)">Automatically create preview URLs for every pull request opened on GitHub.</p>
						<div class="rounded-lg border p-4 font-mono text-xs flex justify-between" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<span>PR #14 — feature/ui-refresh</span>
							<span class="font-bold" style="color: var(--primary)">https://pr-14.devpanel.local</span>
						</div>
					</div>
				{/if}

				{#if activeTab === 'redirects'}
					<div class="card-surface p-6 space-y-4">
						<h2 class="font-bold text-base flex items-center gap-2">
							<span class="material-symbols-outlined text-primary">router</span> Redirects &amp; Rewrites
						</h2>
						<p class="text-xs" style="color: var(--on-surface-variant)">Configure URL redirects and SPA routing fallbacks.</p>
						<div class="rounded-lg border p-4 font-mono text-xs flex justify-between" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<span>/* -&gt; /index.html (200 SPA rewrite)</span>
							<span style="color: var(--success)">Active</span>
						</div>
					</div>
				{/if}

				{#if activeTab === 'headers'}
					<div class="card-surface p-6 space-y-4">
						<h2 class="font-bold text-base flex items-center gap-2">
							<span class="material-symbols-outlined text-primary">view_headline</span> Custom HTTP Headers
						</h2>
						<p class="text-xs" style="color: var(--on-surface-variant)">Inject HTTP response headers for security, CORS, and caching.</p>
						<div class="rounded-lg border p-4 font-mono text-xs space-y-2" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<div>Cache-Control: public, max-age=31536000</div>
							<div>Access-Control-Allow-Origin: *</div>
						</div>
					</div>
				{/if}

				{#if activeTab === 'domains'}
					<div class="card-surface p-6 space-y-6">
						<h2 class="font-bold text-base">Custom Domains &amp; SSL Certificates</h2>
						<form onsubmit={(e) => { e.preventDefault(); handleSaveService(currentSvc!); }} class="space-y-4 max-w-md">
							<div>
								<label for="stDomainInput" class="label-caps block text-xs mb-1">Custom Domain</label>
								<input id="stDomainInput" type="text" bind:value={currentSvc.custom_domain} placeholder="e.g. www.mysite.com" class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
							</div>
							<button type="submit" class="flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 16px">add</span>Attach Custom Domain
							</button>
						</form>
					</div>
				{/if}

				{#if activeTab === 'environment'}
					<div class="card-surface p-6">
						<EnvVarEditor bind:envVars={currentSvc.env_vars} serviceName={currentSvc.name} />
						<div class="mt-4 flex justify-end">
							<button onclick={() => handleSaveService(currentSvc!)} class="flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 18px">save</span>Save &amp; Rebuild
							</button>
						</div>
					</div>
				{/if}

				{#if activeTab === 'logs'}
					<div class="h-[500px]">
						<Terminal projectId={projectId} serviceFilter={currentSvc.name} sourceUrl={sseUrl} title={`${currentSvc.name} — static build logs`} />
					</div>
				{/if}
			</div>

		<!-- ========================================================================= -->
		<!-- SERVICE VIEW 3: WEB SERVICE / GENERAL (web_service_settings_cloudstack)  -->
		<!-- ========================================================================= -->
		{:else}
			<div class="space-y-6">
				<!-- Header section -->
				<div class="flex flex-wrap items-end justify-between gap-4">
					<div>
						<div class="flex items-center gap-3">
							<span class="flex h-12 w-12 items-center justify-center rounded-xl shadow-sm" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 28px">dns</span>
							</span>
							<div>
								<div class="flex items-center gap-2">
									<h1 class="text-2xl font-bold lg:text-3xl" style="color: var(--on-surface)">{currentSvc.name}</h1>
									<span class="rounded-full px-3 py-0.5 text-xs font-bold uppercase" style="background-color: var(--success-container); color: var(--on-success-container)">
										{serviceMetrics.status}
									</span>
								</div>
								<p class="mt-0.5 text-xs" style="color: var(--on-surface-variant)">Web Service Container · Port {currentSvc.port || 8080}</p>
							</div>
						</div>
					</div>

					<div class="flex flex-wrap gap-2">
						<a href={getServiceUrl(projectData?.blueprint?.name || projectId, currentSvc.name, $routingConfig)} target="_blank" rel="noreferrer" class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors hover:opacity-80" style="border-color: var(--outline-variant); background-color: var(--surface-lowest);">
							<span class="material-symbols-outlined" style="font-size: 18px">open_in_new</span>Visit Live Site
						</a>
						<button onclick={() => handleRestart(currentSvc!.name)} class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors hover:opacity-80" style="border-color: var(--outline-variant); background-color: var(--surface-lowest);">
							<span class="material-symbols-outlined" style="font-size: 18px">restart_alt</span>Restart
						</button>
						<button onclick={() => triggerDeploy(projectId)} class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-opacity hover:opacity-90" style="background-color: var(--primary); color: var(--on-primary);">
							<span class="material-symbols-outlined" style="font-size: 18px">rocket_launch</span>Manual Redeploy
						</button>
					</div>
				</div>

				{#if activeTab === 'events' || activeTab === 'overview'}
					<!-- Events audit log from api_main_events_cloudstack -->
					<div class="card-surface p-6 space-y-6">
						<div class="flex items-center justify-between border-b pb-4" style="border-color: var(--outline-variant)">
							<h2 class="font-bold text-base">Service Audit Events</h2>
							<span class="label-caps text-xs" style="color: var(--on-surface-variant)">Showing recent deployments &amp; status triggers</span>
						</div>
						<div class="space-y-3">
							{#each auditEvents as evt}
								<div class="flex items-start gap-4 rounded-lg border p-4 transition-colors" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
									<span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg" style="background-color: var(--surface-low); color: {evt.color}">
										<span class="material-symbols-outlined" style="font-size: 20px">{evt.icon}</span>
									</span>
									<div class="min-w-0 flex-1">
										<div class="flex items-center justify-between">
											<h3 class="font-semibold text-sm">{evt.title}</h3>
											<span class="label-caps text-[10px]" style="color: var(--on-surface-variant)">{evt.time}</span>
										</div>
										<p class="mt-1 text-xs" style="color: var(--on-surface-variant)">Triggered by {evt.user} · Event #{evt.id}</p>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				{#if activeTab === 'settings'}
					<!-- Web service settings layout from web_service_settings_cloudstack (Without Region/Runtime) -->
					<form onsubmit={(e) => { e.preventDefault(); handleSaveService(currentSvc!); }} class="card-surface p-6 space-y-8 max-w-4xl">
						<div class="border-b pb-4" style="border-color: var(--outline-variant)">
							<h2 class="font-bold text-base">Web Service Configuration</h2>
							<p class="text-xs" style="color: var(--on-surface-variant)">Configure instance sizing, build scripts, networking, and execution parameters.</p>
						</div>

						<!-- Section 1: General & Instance Sizing -->
						<div class="space-y-6">
							<div>
								<label for="webSvcName" class="label-caps block text-xs mb-1">Service Name</label>
								<input id="webSvcName" type="text" bind:value={currentSvc.name} class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
							</div>

							<!-- Instance Sizing Cards from web_service_settings_cloudstack -->
							<div>
								<span id="instance-type-label" class="label-caps block text-xs mb-2" style="color: var(--on-surface-variant)">Instance Type Sizing</span>
								<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
									<button type="button" onclick={() => (selectedInstanceType = 'shared')} class="p-4 border-2 rounded-lg text-left transition-all" style={selectedInstanceType === 'shared' ? 'border-color: var(--primary); background-color: var(--surface-low);' : 'border-color: var(--outline-variant); background-color: var(--surface-lowest);'}>
										<div class="flex justify-between items-start mb-1">
											<span class="font-bold text-sm">Shared CPU</span>
											{#if selectedInstanceType === 'shared'}<span class="text-[10px] bg-primary text-white px-1.5 py-0.5 rounded font-bold uppercase">Active</span>{/if}
										</div>
										<p class="text-xs text-on-surface-variant">512MB RAM, 0.1 vCPU</p>
										<p class="text-xs font-bold mt-2 text-primary">$7.00 / mo</p>
									</button>

									<button type="button" onclick={() => (selectedInstanceType = 'standard')} class="p-4 border-2 rounded-lg text-left transition-all" style={selectedInstanceType === 'standard' ? 'border-color: var(--primary); background-color: var(--surface-low);' : 'border-color: var(--outline-variant); background-color: var(--surface-lowest);'}>
										<div class="flex justify-between items-start mb-1">
											<span class="font-bold text-sm">Standard</span>
											{#if selectedInstanceType === 'standard'}<span class="text-[10px] bg-primary text-white px-1.5 py-0.5 rounded font-bold uppercase">Active</span>{/if}
										</div>
										<p class="text-xs text-on-surface-variant">2GB RAM, 1 vCPU</p>
										<p class="text-xs font-bold mt-2 text-primary">$24.00 / mo</p>
									</button>

									<button type="button" onclick={() => (selectedInstanceType = 'performance')} class="p-4 border-2 rounded-lg text-left transition-all" style={selectedInstanceType === 'performance' ? 'border-color: var(--primary); background-color: var(--surface-low);' : 'border-color: var(--outline-variant); background-color: var(--surface-lowest);'}>
										<div class="flex justify-between items-start mb-1">
											<span class="font-bold text-sm">Performance</span>
											{#if selectedInstanceType === 'performance'}<span class="text-[10px] bg-primary text-white px-1.5 py-0.5 rounded font-bold uppercase">Active</span>{/if}
										</div>
										<p class="text-xs text-on-surface-variant">8GB RAM, 4 vCPU</p>
										<p class="text-xs font-bold mt-2 text-primary">$96.00 / mo</p>
									</button>
								</div>
							<!-- Runtime Engine Selector from web_service_settings_cloudstack -->
							<div class="border-t pt-6 space-y-3" style="border-color: var(--outline-variant)">
								<div class="flex items-center gap-2">
									<span class="material-symbols-outlined text-primary text-base">terminal</span>
									<h3 class="font-bold text-xs uppercase tracking-wider" style="color: var(--on-surface-variant)">Runtime Engine</h3>
								</div>
								<div>
									<label id="web-runtime-label" class="block text-xs font-bold mb-2" style="color: var(--on-surface)">Select Language / Runtime</label>
									<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
										{#each [
											{ id: 'Node.js', label: 'Node.js', icon: 'data_object', desc: 'Node 22 LTS' },
											{ id: 'Python 3', label: 'Python 3', icon: 'psychology', desc: 'Python 3.11' },
											{ id: 'Go', label: 'Go', icon: 'speed', desc: 'Go 1.22' },
											{ id: 'Rust', label: 'Rust', icon: 'settings_suggest', desc: 'Rust 1.77' },
											{ id: 'Ruby', label: 'Ruby', icon: 'diamond', desc: 'Ruby 3.3' },
											{ id: 'Elixir', label: 'Elixir', icon: 'water_drop', desc: 'Elixir 1.16' },
											{ id: 'Docker', label: 'Docker', icon: 'developer_board', desc: 'Custom Dockerfile' }
										] as rt}
											<button
												type="button"
												onclick={() => (currentSvc.runtime = rt.id)}
												class="p-3 border-2 rounded-lg text-left transition-all flex flex-col gap-1"
												style={(currentSvc.runtime || 'Node.js') === rt.id
													? 'border-color: var(--primary); background-color: var(--surface-low);'
													: 'border-color: var(--outline-variant); background-color: var(--surface-lowest);'}
											>
												<div class="flex items-center gap-2">
													<span class="material-symbols-outlined text-base" style={(currentSvc.runtime || 'Node.js') === rt.id ? 'color: var(--primary)' : 'color: var(--on-surface-variant)'}>{rt.icon}</span>
													<span class="font-bold text-xs">{rt.label}</span>
												</div>
												<span class="text-[10px]" style="color: var(--on-surface-variant)">{rt.desc}</span>
											</button>
										{/each}
									</div>
								</div>
							</div>
						</div>

						<!-- Section 2: Build & Deploy Commands Grid -->
						<div class="border-t pt-6 space-y-4" style="border-color: var(--outline-variant)">
							<h3 class="font-bold text-sm">Build &amp; Deployment Execution</h3>
							<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
								<div>
									<label for="webSvcPort" class="label-caps block text-xs mb-1">Target Container Port</label>
									<input id="webSvcPort" type="number" bind:value={currentSvc.port} class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
								</div>
								<div>
									<label for="webSvcDomain" class="label-caps block text-xs mb-1">Custom Domain</label>
									<input id="webSvcDomain" type="text" bind:value={currentSvc.custom_domain} placeholder="api.company.com" class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
								</div>
							</div>

							<div>
								<label for="webSvcBuild" class="label-caps block text-xs mb-1">Build Command</label>
								<input id="webSvcBuild" type="text" bind:value={currentSvc.build_command} placeholder="npm ci && npm run build" class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
							</div>

							<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
								<div>
									<label for="webPreDeploy" class="label-caps block text-xs mb-1">Pre-Deploy Command</label>
									<input id="webPreDeploy" type="text" bind:value={preDeployCommand} placeholder="npm run migrate" class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
								</div>
								<div>
									<label for="webSvcStart" class="label-caps block text-xs mb-1">Start Command</label>
									<input id="webSvcStart" type="text" bind:value={currentSvc.start_command} placeholder="npm start" class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
								</div>
							</div>

							<!-- Deploy Hook URL -->
							<div>
								<span class="label-caps block text-xs mb-1" style="color: var(--on-surface-variant)">Deploy Hook URL</span>
								<div class="flex items-center gap-2 rounded-lg border p-2 font-mono text-xs" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
									<span class="flex-1 truncate">{`http://localhost:8090/api/deploy-hook/${projectId}/${currentSvc.name}`}</span>
									<button type="button" onclick={() => copyToClipboard(`http://localhost:8090/api/deploy-hook/${projectId}/${currentSvc.name}`)} class="text-primary hover:opacity-80">
										<span class="material-symbols-outlined" style="font-size: 18px">content_copy</span>
									</button>
								</div>
							</div>
						</div>

						<!-- Section 3: EXPANDED Additional Configuration Grid (All Options Present) -->
						<div class="border-t pt-6 space-y-4" style="border-color: var(--outline-variant)">
							<h3 class="font-bold text-sm">Additional Configuration</h3>
							<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
								<button type="button" onclick={() => (activeTab = 'previews')} class="rounded-xl border p-4 text-left shadow-sm hover:border-[color:var(--primary)] transition-all" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
									<div class="flex h-9 w-9 items-center justify-center rounded-lg mb-2" style="background-color: var(--primary-fixed); color: var(--primary)">
										<span class="material-symbols-outlined" style="font-size: 20px">visibility</span>
									</div>
									<h4 class="font-bold text-xs mb-1">PR Previews</h4>
									<p class="text-[11px]" style="color: var(--on-surface-variant)">Isolated environments for every pull request.</p>
									<span class="mt-2 inline-block rounded px-2 py-0.5 text-[10px] font-bold uppercase" style="background-color: var(--primary-fixed); color: var(--primary)">Enabled</span>
								</button>

								<button type="button" onclick={() => (activeTab = 'shell')} class="rounded-xl border p-4 text-left shadow-sm hover:border-[color:var(--primary)] transition-all" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
									<div class="flex h-9 w-9 items-center justify-center rounded-lg mb-2" style="background-color: var(--surface-low); color: var(--on-surface-variant)">
										<span class="material-symbols-outlined" style="font-size: 20px">terminal</span>
									</div>
									<h4 class="font-bold text-xs mb-1">Container Shell</h4>
									<p class="text-[11px]" style="color: var(--on-surface-variant)">Interactive terminal shell inside container.</p>
									<span class="mt-2 inline-block rounded px-2 py-0.5 text-[10px] font-bold uppercase" style="background-color: var(--surface-high)">Open Shell</span>
								</button>

								<button type="button" onclick={() => (activeTab = 'scaling')} class="rounded-xl border p-4 text-left shadow-sm hover:border-[color:var(--primary)] transition-all" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
									<div class="flex h-9 w-9 items-center justify-center rounded-lg mb-2" style="background-color: var(--surface-low); color: var(--on-surface-variant)">
										<span class="material-symbols-outlined" style="font-size: 20px">dynamic_form</span>
									</div>
									<h4 class="font-bold text-xs mb-1">Autoscaling</h4>
									<p class="text-[11px]" style="color: var(--on-surface-variant)">Auto-scale replicas based on CPU/RAM load.</p>
									<span class="mt-2 inline-block rounded px-2 py-0.5 text-[10px] font-bold uppercase" style="background-color: var(--surface-high)">1 Instance</span>
								</button>

								<div class="rounded-xl border p-4 shadow-sm hover:border-[color:var(--primary)] transition-all" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
									<div class="flex h-9 w-9 items-center justify-center rounded-lg mb-2" style="background-color: var(--surface-low); color: var(--on-surface-variant)">
										<span class="material-symbols-outlined" style="font-size: 20px">speed</span>
									</div>
									<h4 class="font-bold text-xs mb-1">Edge Caching</h4>
									<p class="text-[11px]" style="color: var(--on-surface-variant)">Global Caddy edge caching network.</p>
									<span class="mt-2 inline-block rounded px-2 py-0.5 text-[10px] font-bold uppercase" style="background-color: var(--surface-high)">Active</span>
								</div>

								<div class="rounded-xl border p-4 shadow-sm hover:border-[color:var(--primary)] transition-all" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
									<div class="flex h-9 w-9 items-center justify-center rounded-lg mb-2" style="background-color: var(--primary-fixed); color: var(--primary)">
										<span class="material-symbols-outlined" style="font-size: 20px">notifications_active</span>
									</div>
									<h4 class="font-bold text-xs mb-1">Notifications</h4>
									<p class="text-[11px]" style="color: var(--on-surface-variant)">Slack, Email &amp; Webhook alerts.</p>
									<span class="mt-2 inline-block rounded px-2 py-0.5 text-[10px] font-bold uppercase" style="background-color: var(--primary-fixed); color: var(--primary)">Active</span>
								</div>

								<button type="button" onclick={() => (activeTab = 'jobs')} class="rounded-xl border p-4 text-left shadow-sm hover:border-[color:var(--primary)] transition-all" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
									<div class="flex h-9 w-9 items-center justify-center rounded-lg mb-2" style="background-color: var(--primary-fixed); color: var(--primary)">
										<span class="material-symbols-outlined" style="font-size: 20px">bolt</span>
									</div>
									<h4 class="font-bold text-xs mb-1">One-Off Jobs</h4>
									<p class="text-[11px]" style="color: var(--on-surface-variant)">Run one-time scripts or db migrations.</p>
									<span class="mt-2 inline-block rounded px-2 py-0.5 text-[10px] font-bold uppercase" style="background-color: var(--primary-fixed); color: var(--primary)">Run Job</span>
								</button>
							</div>
						</div>

						<!-- Section 4: Maintenance Mode & Danger Zone -->
						<div class="border-t pt-6 space-y-4" style="border-color: var(--outline-variant)">
							<div class="rounded-xl border p-5 flex items-center justify-between" style="border-color: var(--warning); background-color: var(--warning-container)">
								<div>
									<h4 class="font-bold text-sm flex items-center gap-2">
										<span class="material-symbols-outlined">warning</span> Maintenance Mode
									</h4>
									<p class="text-xs mt-1" style="color: var(--on-surface-variant)">Take service offline for critical updates. Incoming traffic receives 503 HTTP status.</p>
								</div>
								<button type="button" onclick={() => (maintenanceMode = !maintenanceMode)} class="rounded-lg border-2 px-4 py-2 text-xs font-bold transition-colors" style={maintenanceMode ? 'background-color: var(--warning); color: white; border-color: var(--warning);' : 'border-color: var(--warning); color: var(--warning);'}>
									{maintenanceMode ? 'Exit Maintenance' : 'Enable Maintenance'}
								</button>
							</div>
						</div>

						<div class="pt-4 border-t flex justify-end" style="border-color: var(--outline-variant)">
							<button type="submit" disabled={isSaving} class="flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 18px">save</span>
								{isSaving ? 'Saving...' : 'Save & Redeploy'}
							</button>
						</div>
					</form>
				{/if}

				<!-- Dedicated Tab Views for Web Service Unique Sidebar Options -->
				{#if activeTab === 'shell'}
					<div class="card-surface p-6 space-y-4">
						<h2 class="font-bold text-base flex items-center gap-2">
							<span class="material-symbols-outlined text-primary">terminal</span> Interactive Web Shell
						</h2>
						<p class="text-xs" style="color: var(--on-surface-variant)">Execute terminal commands directly inside the running container.</p>
						<div class="h-[400px]">
							<Terminal projectId={projectId} serviceFilter={currentSvc.name} sourceUrl={sseUrl} title={`${currentSvc.name} — sh / bash shell`} />
						</div>
					</div>
				{/if}

				{#if activeTab === 'scaling'}
					<div class="card-surface p-6 space-y-6">
						<h2 class="font-bold text-base flex items-center gap-2">
							<span class="material-symbols-outlined text-primary">dynamic_form</span> Container Scaling &amp; Compute
						</h2>
						<div class="space-y-4 max-w-md">
							<div>
								<label for="replicaInput" class="label-caps block text-xs mb-1">Replica Instance Count</label>
								<input id="replicaInput" type="number" min="1" max="10" bind:value={replicaCount} class="w-full rounded border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
							</div>
							<button onclick={() => (saveMessage = `Scaled ${currentSvc!.name} to ${replicaCount} container replicas.`)} class="flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 16px">tune</span>Apply Replica Scaling
							</button>
						</div>
					</div>
				{/if}

				{#if activeTab === 'previews'}
					<div class="card-surface p-6 space-y-4">
						<h2 class="font-bold text-base flex items-center gap-2">
							<span class="material-symbols-outlined text-primary">visibility</span> Pull Request Previews
						</h2>
						<p class="text-xs" style="color: var(--on-surface-variant)">Isolated preview containers deployed automatically for PR branch builds.</p>
						<div class="rounded-lg border p-4 font-mono text-xs flex justify-between" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<span>PR #28 — api/v2-refactor</span>
							<span class="font-bold text-primary">https://pr-28-api.devpanel.local</span>
						</div>
					</div>
				{/if}

				{#if activeTab === 'jobs'}
					<div class="card-surface p-6 space-y-4">
						<h2 class="font-bold text-base flex items-center gap-2">
							<span class="material-symbols-outlined text-primary">bolt</span> One-Off Jobs &amp; Exec Tasks
						</h2>
						<p class="text-xs" style="color: var(--on-surface-variant)">Run one-off database migrations, seed scripts, or maintenance commands.</p>
						<div class="flex gap-2">
							<input type="text" placeholder="e.g. npx prisma db push" class="flex-1 rounded-lg border p-2.5 font-mono text-xs" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
							<button onclick={() => (saveMessage = 'Executed one-off job successfully.')} class="rounded-lg px-4 py-2 text-xs font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
								Run Command
							</button>
						</div>
					</div>
				{/if}

				{#if activeTab === 'environment'}
					<div class="card-surface p-6">
						<EnvVarEditor bind:envVars={currentSvc.env_vars} serviceName={currentSvc.name} />
						<div class="mt-4 flex justify-end">
							<button onclick={() => handleSaveService(currentSvc!)} class="flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 18px">save</span>Save &amp; Apply
							</button>
						</div>
					</div>
				{/if}

				{#if activeTab === 'logs'}
					<div class="h-[500px]">
						<Terminal projectId={projectId} serviceFilter={currentSvc.name} sourceUrl={sseUrl} title={`${currentSvc.name} — live container logs`} />
					</div>
				{/if}

				{#if activeTab === 'domains'}
					<div class="card-surface p-6 space-y-6">
						<h2 class="font-bold text-base">Custom Domains &amp; SSL Certificates</h2>
						<form onsubmit={(e) => { e.preventDefault(); handleSaveService(currentSvc!); }} class="space-y-4 max-w-md">
							<div>
								<label for="webDomainAdd" class="label-caps block text-xs mb-1">Custom Domain</label>
								<input id="webDomainAdd" type="text" bind:value={currentSvc.custom_domain} placeholder="e.g. api.mydomain.com" class="w-full rounded-lg border p-2.5 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)" />
							</div>
							<button type="submit" class="flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined" style="font-size: 16px">add</span>Attach Custom Domain
							</button>
						</form>
					</div>
				{/if}

				{#if activeTab === 'volumes'}
					<div class="card-surface p-6 space-y-4">
						<h2 class="font-bold text-base">Mounted Storage Disks</h2>
						<div class="rounded border p-4 font-mono text-xs flex justify-between" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<span>/var/lib/{currentSvc.name}/data</span>
							<span style="color: var(--success)">Mounted</span>
						</div>
					</div>
				{/if}

				{#if activeTab === 'metrics'}
					<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
						<div class="card-surface p-5">
							<span class="label-caps" style="color: var(--on-surface-variant)">CPU Utilisation</span>
							<div class="my-2 text-2xl font-bold">{serviceMetrics.cpuPercent.toFixed(1)}%</div>
							<div class="h-2 w-full rounded-full" style="background-color: var(--surface-high)">
								<div class="h-2 rounded-full transition-all" style="width: {serviceMetrics.cpuPercent}%; background-color: var(--primary)"></div>
							</div>
						</div>

						<div class="card-surface p-5">
							<span class="label-caps" style="color: var(--on-surface-variant)">Memory Usage</span>
							<div class="my-2 text-2xl font-bold">{Math.round(serviceMetrics.memoryMb)} MB</div>
							<div class="h-2 w-full rounded-full" style="background-color: var(--surface-high)">
								<div class="h-2 rounded-full transition-all" style="width: {Math.min((serviceMetrics.memoryMb / 512) * 100, 100)}%; background-color: var(--primary)"></div>
							</div>
						</div>
					</div>
				{/if}
			</div>
		{/if}

	{:else if !projectData}
		<div class="flex flex-col items-center justify-center py-20 text-center">
			<span class="material-symbols-outlined animate-spin text-4xl mb-3" style="color: var(--primary)">sync</span>
			<p class="font-semibold text-sm" style="color: var(--on-surface-variant)">Loading project configuration &amp; build status...</p>
		</div>
	{:else}
		<div class="card-surface p-8 text-center space-y-4 max-w-lg mx-auto my-12">
			<span class="material-symbols-outlined text-4xl" style="color: var(--warning)">warning</span>
			<h2 class="font-bold text-base">No services configured for this project yet.</h2>
			<p class="text-xs" style="color: var(--on-surface-variant)">Click below to launch initial build and deploy services.</p>
			<button onclick={() => triggerDeploy(projectId)} class="rounded-lg px-4 py-2 text-sm font-semibold transition-opacity hover:opacity-90" style="background-color: var(--primary); color: var(--on-primary);">
				Trigger Project Build &amp; Deploy
			</button>
		</div>
	{/if}
</AppShell>
