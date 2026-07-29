<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import EnvVarEditor from '$lib/components/EnvVarEditor.svelte';
	import Terminal from '$lib/components/Terminal.svelte';
	import { getProject, updateService, triggerDeploy, restartService } from '$lib/api';
	import type { ProjectDetail, ServiceRecord } from '$lib/types';

	let projectId = $derived(page.params.id || '');

	let projectData = $state<ProjectDetail | null>(null);
	let activeServiceIdx = $state(0);
	let activeSubTab = $state<'logs' | 'environment' | 'settings' | 'metrics'>('logs');
	let logType = $state<'build' | 'live' | '1h' | '24h'>('build');
	let currentSvc = $derived(projectData?.services?.[activeServiceIdx]);
	let sseUrl = $derived.by(() => {
		if (logType === 'build') return `/api/projects/${projectId}/logs`;
		if (!currentSvc) return '';
		if (logType === 'live') return `/api/projects/${projectId}/services/${currentSvc.name}/logs`;
		if (logType === '1h') return `/api/projects/${projectId}/services/${currentSvc.name}/logs?since=1h`;
		if (logType === '24h') return `/api/projects/${projectId}/services/${currentSvc.name}/logs?since=24h`;
		return '';
	});

	let isSaving = $state(false);
	let saveMessage = $state<string | null>(null);

	// Live Metrics State
	let serviceMetrics = $state({
		cpuPercent: 0,
		memoryMb: 0,
		status: 'Healthy'
	});
	let metricsInterval: any = null;

	async function loadProjectData() {
		if (!projectId) return;
		try {
			const res = await getProject(projectId);
			projectData = res;
		} catch (e) {
			console.error('Failed to load project details:', e);
		}
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
				start_command: svc.start_command
			});
			saveMessage = `Saved settings for service ${svc.name}! Triggering redeployment...`;
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
		if (!projectData || !projectData.services[activeServiceIdx]) return;
		const currentSvc = projectData.services[activeServiceIdx];
		try {
			const res = await fetch('/api/containers');
			if (!res.ok) return;
			const data = await res.json();
			const list = data.containers || [];

			const targetName = `devpnl-${projectId.replace(/^bp-/, '')}-${currentSvc.name}`;
			const container = list.find((c: any) => c.name === targetName || c.name.includes(currentSvc.name));

			if (container) {
				serviceMetrics = {
					cpuPercent: container.cpuPercent || 0,
					memoryMb: container.memoryMb || 0,
					status: container.status === 'running' ? 'Healthy' : 'Stopped'
				};
			} else {
				serviceMetrics = { cpuPercent: 0, memoryMb: 0, status: 'Unknown' };
			}
		} catch (e) {
			console.error('Metrics error:', e);
		}
	}

	onMount(() => {
		loadProjectData();
		metricsInterval = setInterval(fetchServiceMetrics, 3000);
	});

	onDestroy(() => {
		if (metricsInterval) clearInterval(metricsInterval);
	});
</script>

<div class="min-h-screen bg-gray-50 text-gray-900 font-sans flex flex-col antialiased">
	<!-- Top Bar Header -->
	<header class="border-b border-gray-200 bg-white py-4 px-6 md:px-10 flex items-center justify-between shadow-sm">
		<div class="flex items-center gap-3">
			<a href="/" aria-label="Back to Dashboard" class="p-2 text-gray-400 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/></svg>
			</a>
			<div>
				<h1 class="text-xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
					<span>{projectData?.blueprint.name || projectId}</span>
					<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-green-100 text-green-700 border border-green-200 uppercase">
						{projectData?.blueprint.status || 'Active'}
					</span>
				</h1>
				<div class="flex items-center gap-3 mt-1">
					<p class="text-xs text-gray-500 font-mono">{projectData?.blueprint.repo_url || 'Application Stack'}</p>
					
					{#if projectData?.services?.some(s => s.type === 'web' || s.type === 'static')}
						<span class="text-gray-300">•</span>
						<a
							href={`/app/${projectData.blueprint.name.toLowerCase()}`}
							target="_blank"
							rel="noreferrer"
							class="text-blue-600 hover:text-blue-700 text-xs flex items-center gap-1 transition-colors font-medium"
						>
							<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/></svg>
							Visit Site
						</a>
					{/if}
				</div>
			</div>
		</div>

		<div class="flex items-center gap-3">
			{#if projectData && projectData.services[activeServiceIdx]}
				<button
					type="button"
					onclick={() => handleRestart(projectData!.services[activeServiceIdx].name)}
					class="bg-white hover:bg-gray-50 text-gray-700 border border-gray-300 px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm"
				>
					↻ Restart Container
				</button>
			{/if}
			<button
				type="button"
				onclick={() => triggerDeploy(projectId)}
				class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm"
			>
				Manual Redeploy
			</button>
		</div>
	</header>

	<!-- Main Workspace -->
	<main class="flex-1 p-6 md:p-10 max-w-7xl mx-auto w-full space-y-6">
		{#if saveMessage}
			<div class="p-4 rounded-xl border text-sm flex items-center justify-between shadow-sm {saveMessage.includes('Failed') ? 'bg-red-50 border-red-200 text-red-700' : 'bg-green-50 border-green-200 text-green-700'}">
				<span>{saveMessage}</span>
				<button type="button" onclick={() => (saveMessage = null)} class="font-bold">✕</button>
			</div>
		{/if}

		{#if projectData}
			<!-- Multi-service tabs if app defines multiple services -->
			{#if projectData.services.length > 1}
				<div class="flex border-b border-gray-200 gap-2">
					{#each projectData.services as svc, idx}
						<button
							type="button"
							onclick={() => { activeServiceIdx = idx; fetchServiceMetrics(); }}
							class="px-4 py-2.5 text-sm font-semibold border-b-2 transition-colors flex items-center gap-2 {activeServiceIdx === idx
								? 'border-blue-600 text-blue-600'
								: 'border-transparent text-gray-500 hover:text-gray-800'}"
						>
							<span>{svc.name}</span>
							<span class="text-xs px-2 py-0.5 rounded-full bg-gray-100 border text-gray-600 uppercase">{svc.type}</span>
						</button>
					{/each}
				</div>
			{/if}

			{@const currentSvc = projectData.services[activeServiceIdx]}

			<!-- View Tabs -->
			<div class="flex border-b border-gray-200 gap-6 text-sm font-medium text-gray-500">
				<button
					type="button"
					onclick={() => (activeSubTab = 'logs')}
					class="py-3 border-b-2 transition-colors flex items-center gap-2 {activeSubTab === 'logs' ? 'border-blue-600 text-blue-600 font-semibold' : 'border-transparent hover:text-gray-900'}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
					<span>Logs Stream</span>
				</button>

				<button
					type="button"
					onclick={() => (activeSubTab = 'environment')}
					class="py-3 border-b-2 transition-colors flex items-center gap-2 {activeSubTab === 'environment' ? 'border-blue-600 text-blue-600 font-semibold' : 'border-transparent hover:text-gray-900'}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/></svg>
					<span>Environment Variables</span>
				</button>

				<button
					type="button"
					onclick={() => (activeSubTab = 'metrics')}
					class="py-3 border-b-2 transition-colors flex items-center gap-2 {activeSubTab === 'metrics' ? 'border-blue-600 text-blue-600 font-semibold' : 'border-transparent hover:text-gray-900'}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
					<span>Metrics Telemetry</span>
				</button>

				<button
					type="button"
					onclick={() => (activeSubTab = 'settings')}
					class="py-3 border-b-2 transition-colors flex items-center gap-2 {activeSubTab === 'settings' ? 'border-blue-600 text-blue-600 font-semibold' : 'border-transparent hover:text-gray-900'}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"/></svg>
					<span>Build & Run Settings</span>
				</button>
			</div>

			<!-- Tab Content Areas -->
			{#if activeSubTab === 'logs'}
				<div class="h-[550px] flex flex-col gap-4">
					<div class="flex items-center justify-between">
						<div>
							<h3 class="text-lg font-semibold text-gray-900">Live Logs & History</h3>
							<p class="text-sm text-gray-500 mt-1">View build progress and real-time container logs.</p>
						</div>
						<select
							bind:value={logType}
							class="bg-white border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block p-2.5 shadow-sm"
						>
							<option value="build">Deployment Build Logs</option>
							<option value="live">Live Container Output</option>
							<option value="1h">Last 1 Hour Container Logs</option>
							<option value="24h">Last 24 Hours Container Logs</option>
						</select>
					</div>
					<div class="flex-1 min-h-0">
						{#key logType}
							<Terminal {projectId} sourceUrl={sseUrl} title={`Log Stream: ${logType === 'build' ? 'Deployment Build' : currentSvc?.name}`} />
						{/key}
					</div>
				</div>
			{:else if activeSubTab === 'environment'}
				<div class="bg-white border border-gray-200 rounded-xl p-6 shadow-sm space-y-6">
					<div>
						<h3 class="text-lg font-semibold text-gray-900">Environment Variables ({currentSvc.name})</h3>
						<p class="text-sm text-gray-500 mt-1">Configure environment variables for {currentSvc.name}.</p>
					</div>

					<EnvVarEditor bind:envVars={currentSvc.env_vars} />

					<div class="flex justify-end">
						<button
							type="button"
							onclick={() => handleSaveService(currentSvc)}
							disabled={isSaving}
							class="bg-blue-600 hover:bg-blue-700 text-white px-5 py-2.5 rounded-lg text-sm font-medium transition-colors shadow-sm disabled:opacity-50"
						>
							{isSaving ? 'Saving Environment...' : 'Save & Redeploy'}
						</button>
					</div>
				</div>
			{:else if activeSubTab === 'metrics'}
				<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
					<div class="bg-white border border-gray-200 rounded-xl p-6 shadow-sm">
						<span class="text-sm font-medium text-gray-500">Service Status</span>
						<p class="text-2xl font-bold text-gray-900 mt-2">{serviceMetrics.status}</p>
					</div>
					<div class="bg-white border border-gray-200 rounded-xl p-6 shadow-sm">
						<span class="text-sm font-medium text-gray-500">CPU Usage</span>
						<p class="text-2xl font-bold text-gray-900 mt-2">{serviceMetrics.cpuPercent}%</p>
					</div>
					<div class="bg-white border border-gray-200 rounded-xl p-6 shadow-sm">
						<span class="text-sm font-medium text-gray-500">Memory Usage</span>
						<p class="text-2xl font-bold text-gray-900 mt-2">{serviceMetrics.memoryMb} MB</p>
					</div>
				</div>
			{:else if activeSubTab === 'settings'}
				<div class="bg-white border border-gray-200 rounded-xl p-6 md:p-8 shadow-sm space-y-6 max-w-3xl">
					<div>
						<h3 class="text-lg font-semibold text-gray-900">Service Build & Runtime Settings</h3>
						<p class="text-sm text-gray-500 mt-1">Edit commands and container target port for {currentSvc.name}.</p>
					</div>

					<div class="space-y-4">
						<div>
							<label for="svcPort" class="block text-sm font-medium text-gray-700 mb-1.5">Target Container Port</label>
							<input id="svcPort" type="number" bind:value={currentSvc.port} class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 font-mono text-sm shadow-sm" />
						</div>

						<div>
							<label for="svcBuildCmd" class="block text-sm font-medium text-gray-700 mb-1.5">Build Command</label>
							<input id="svcBuildCmd" type="text" bind:value={currentSvc.build_command} class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 font-mono text-sm shadow-sm" />
						</div>

						<div>
							<label for="svcStartCmd" class="block text-sm font-medium text-gray-700 mb-1.5">Start / Execute Command</label>
							<input id="svcStartCmd" type="text" bind:value={currentSvc.start_command} class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 font-mono text-sm shadow-sm" />
						</div>

						<div>
							<label for="svcDomain" class="block text-sm font-medium text-gray-700 mb-1.5">Custom Routing Domain</label>
							<input id="svcDomain" type="text" bind:value={currentSvc.custom_domain} placeholder="e.g. app.example.com" class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 text-sm shadow-sm" />
						</div>

						<div class="pt-4 flex justify-end">
							<button
								type="button"
								onclick={() => handleSaveService(currentSvc)}
								disabled={isSaving}
								class="bg-blue-600 hover:bg-blue-700 text-white px-5 py-2.5 rounded-lg text-sm font-medium transition-colors shadow-sm disabled:opacity-50"
							>
								{isSaving ? 'Saving...' : 'Save Service Settings'}
							</button>
						</div>
					</div>
				</div>
			{/if}
		{:else}
			<div class="p-12 text-center text-gray-500 bg-white border border-gray-200 rounded-xl shadow-sm">
				<p class="font-medium">Loading project details for {projectId}...</p>
			</div>
		{/if}
	</main>
</div>
