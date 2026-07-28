<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import Terminal from '$lib/components/Terminal.svelte';
	import EnvVarEditor from '$lib/components/EnvVarEditor.svelte';
	import { getProject, updateService, triggerDeploy, restartService } from '$lib/api';
	import type { ProjectDetail, ServiceRecord } from '$lib/types';

	let projectId = $derived(page.params.id || '');

	let projectData = $state<ProjectDetail | null>(null);
	let activeServiceIdx = $state(0);
	let activeSubTab = $state<'logs' | 'environment' | 'settings' | 'metrics'>('logs');

	let isSaving = $state(false);
	let saveMessage = $state<string | null>(null);

	// Metrics state
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

<div class="flex h-screen bg-neutral-950 text-neutral-100 font-sans antialiased overflow-hidden">
	{#if projectData}
		<!-- Left Sidebar Service Navigation -->
		<aside class="w-64 border-r border-neutral-800 bg-neutral-900/70 backdrop-blur-md flex flex-col justify-between p-4 shrink-0">
			<div class="space-y-6">
				<!-- Project Header -->
				<div class="px-3 py-2 border-b border-neutral-800/80 pb-4">
					<h2 class="font-bold text-base text-neutral-100 truncate font-mono" title={projectData.blueprint.name}>
						{projectData.blueprint.name}
					</h2>
					<div class="flex items-center gap-2 mt-1">
						<span class="px-2 py-0.5 rounded-full text-[10px] font-semibold uppercase tracking-wider font-mono border {projectData.blueprint.status === 'active'
							? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
							: 'bg-amber-500/10 text-amber-400 border-amber-500/20'}">
							{projectData.blueprint.status}
						</span>
					</div>
				</div>

				<!-- Service Tabs Navigation -->
				<div class="space-y-1">
					<span class="text-[10px] uppercase font-bold text-neutral-500 tracking-wider px-3 block mb-2">Services</span>
					{#each projectData.services as svc, idx}
						<button
							onclick={() => {
								activeServiceIdx = idx;
								fetchServiceMetrics();
							}}
							class="w-full flex items-center justify-between px-3.5 py-2.5 rounded-xl text-xs font-mono font-medium transition-all {activeServiceIdx === idx
								? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
								: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
						>
							<div class="flex items-center gap-2 truncate">
								<span class="w-2 h-2 rounded-full {svc.type === 'static' ? 'bg-sky-400' : svc.type === 'web' ? 'bg-purple-400' : 'bg-emerald-400'}"></span>
								<span class="truncate">{svc.name}</span>
							</div>
							<span class="text-[10px] uppercase text-neutral-500">{svc.type}</span>
						</button>
					{/each}
				</div>
			</div>

			<div class="pt-4 border-t border-neutral-800">
				<a href="/" class="text-xs font-medium text-neutral-400 hover:text-emerald-400 transition-colors flex items-center gap-2">
					← Back to Dashboard
				</a>
			</div>
		</aside>

		<!-- Main Workspace for Selected Service -->
		{#if projectData.services[activeServiceIdx]}
			{@const currentSvc = projectData.services[activeServiceIdx]}
			<main class="flex-1 flex flex-col min-w-0 bg-neutral-950 p-6 md:p-8 overflow-y-auto">
				<!-- Top Header -->
				<div class="flex items-center justify-between border-b border-neutral-800/80 pb-4 mb-6">
					<div>
						<div class="flex items-center gap-3">
							<h1 class="text-2xl font-bold text-neutral-100 font-mono">{currentSvc.name}</h1>
							<span class="px-2.5 py-0.5 rounded-full text-xs font-mono uppercase font-bold border border-neutral-800 bg-neutral-900 text-neutral-300">
								{currentSvc.type}
							</span>
						</div>
						<p class="text-xs text-neutral-400 font-mono mt-1">
							Hosted URL: <a href="/app/{projectData.blueprint.name.toLowerCase()}" target="_blank" class="text-emerald-400 hover:underline">{typeof window !== 'undefined' ? window.location.origin : ''}/app/{projectData.blueprint.name.toLowerCase()} ↗</a>
						</p>
					</div>

					<div class="flex items-center gap-3">
						<button
							onclick={() => handleRestart(currentSvc.name)}
							class="px-3.5 py-2 rounded-xl bg-neutral-900 border border-neutral-800 hover:border-neutral-700 text-neutral-300 text-xs font-mono transition-all"
						>
							↻ Restart Container
						</button>
						<button
							onclick={() => triggerDeploy(projectId)}
							class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-bold transition-all shadow-lg"
						>
							Manual Redeploy
						</button>
					</div>
				</div>

				<!-- Save Message Toast -->
				{#if saveMessage}
					<div class="p-4 rounded-xl bg-neutral-900 border border-neutral-800 text-emerald-400 text-xs font-mono mb-6 flex items-center justify-between">
						<span>{saveMessage}</span>
						<button onclick={() => (saveMessage = null)} class="text-neutral-400 hover:text-neutral-200">✕</button>
					</div>
				{/if}

				<!-- Subtab Selector -->
				<div class="flex items-center gap-2 border-b border-neutral-800/80 pb-3 mb-6">
					<button
						onclick={() => (activeSubTab = 'logs')}
						class="px-4 py-2 rounded-xl text-xs font-mono font-bold transition-all {activeSubTab === 'logs'
							? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
							: 'text-neutral-400 hover:text-neutral-200'}"
					>
						Logs Stream
					</button>
					<button
						onclick={() => (activeSubTab = 'environment')}
						class="px-4 py-2 rounded-xl text-xs font-mono font-bold transition-all {activeSubTab === 'environment'
							? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
							: 'text-neutral-400 hover:text-neutral-200'}"
					>
						Environment Variables
					</button>
					<button
						onclick={() => (activeSubTab = 'settings')}
						class="px-4 py-2 rounded-xl text-xs font-mono font-bold transition-all {activeSubTab === 'settings'
							? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
							: 'text-neutral-400 hover:text-neutral-200'}"
					>
						Build & Run Settings
					</button>
					<button
						onclick={() => (activeSubTab = 'metrics')}
						class="px-4 py-2 rounded-xl text-xs font-mono font-bold transition-all {activeSubTab === 'metrics'
							? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
							: 'text-neutral-400 hover:text-neutral-200'}"
					>
						Metrics
					</button>
				</div>

				<!-- SUBTAB 1: Service Filtered Logs -->
				{#if activeSubTab === 'logs'}
					<div class="flex-1 min-h-[450px]">
						<Terminal projectId={projectId} serviceFilter={currentSvc.name} title={`Live Logs for ${currentSvc.name}`} />
					</div>
				{/if}

				<!-- SUBTAB 2: Environment Variables -->
				{#if activeSubTab === 'environment'}
					<div class="max-w-3xl space-y-6">
						<div class="p-6 rounded-2xl bg-neutral-900/60 border border-neutral-800 space-y-4">
							<h3 class="text-base font-bold text-neutral-100">Environment Variables ({currentSvc.name})</h3>
							<EnvVarEditor bind:envVars={currentSvc.env_vars} />
						</div>

						<div class="flex justify-end">
							<button
								onclick={() => handleSaveService(currentSvc)}
								disabled={isSaving}
								class="px-6 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs transition-all shadow-lg disabled:opacity-50"
							>
								{isSaving ? 'Saving...' : 'Save & Redeploy'}
							</button>
						</div>
					</div>
				{/if}

				<!-- SUBTAB 3: Settings -->
				{#if activeSubTab === 'settings'}
					<div class="max-w-3xl space-y-6">
						<div class="p-6 rounded-2xl bg-neutral-900/60 border border-neutral-800 space-y-4">
							<h3 class="text-base font-bold text-neutral-100 font-mono">Service Configuration ({currentSvc.name})</h3>

							<div class="grid grid-cols-2 gap-4 text-xs font-mono">
								<div class="space-y-1">
									<label for="portInput" class="text-neutral-400 block">Exposed Port</label>
									<input
										id="portInput"
										type="number"
										bind:value={currentSvc.port}
										class="w-full px-3 py-2 bg-neutral-950 border border-neutral-800 rounded-xl text-neutral-100"
									/>
								</div>

								<div class="space-y-1">
									<label for="domInput" class="text-neutral-400 block">Custom Domain</label>
									<input
										id="domInput"
										type="text"
										bind:value={currentSvc.custom_domain}
										placeholder="api.example.com"
										class="w-full px-3 py-2 bg-neutral-950 border border-neutral-800 rounded-xl text-neutral-100"
									/>
								</div>
							</div>

							<div class="space-y-1 pt-2">
								<label for="bCmd" class="text-neutral-400 block text-xs font-mono">Build Command</label>
								<input
									id="bCmd"
									type="text"
									bind:value={currentSvc.build_command}
									class="w-full px-3 py-2 bg-neutral-950 border border-neutral-800 rounded-xl text-neutral-100 text-xs font-mono"
								/>
							</div>

							<div class="space-y-1 pt-2">
								<label for="rCmd" class="text-neutral-400 block text-xs font-mono">Start Command</label>
								<input
									id="rCmd"
									type="text"
									bind:value={currentSvc.start_command}
									class="w-full px-3 py-2 bg-neutral-950 border border-neutral-800 rounded-xl text-neutral-100 text-xs font-mono"
								/>
							</div>
						</div>

						<div class="flex justify-end">
							<button
								onclick={() => handleSaveService(currentSvc)}
								disabled={isSaving}
								class="px-6 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs transition-all shadow-lg disabled:opacity-50"
							>
								{isSaving ? 'Saving...' : 'Save Settings'}
							</button>
						</div>
					</div>
				{/if}

				<!-- SUBTAB 4: Metrics -->
				{#if activeSubTab === 'metrics'}
					<div class="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-4xl">
						<div class="p-5 rounded-2xl bg-neutral-900/60 border border-neutral-800 space-y-2">
							<span class="text-xs text-neutral-400 font-medium">CPU Load ({currentSvc.name})</span>
							<div class="text-2xl font-bold text-neutral-100 font-mono">{serviceMetrics.cpuPercent}%</div>
						</div>

						<div class="p-5 rounded-2xl bg-neutral-900/60 border border-neutral-800 space-y-2">
							<span class="text-xs text-neutral-400 font-medium">Memory Usage</span>
							<div class="text-2xl font-bold text-neutral-100 font-mono">{serviceMetrics.memoryMb} MB</div>
						</div>

						<div class="p-5 rounded-2xl bg-neutral-900/60 border border-neutral-800 space-y-2">
							<span class="text-xs text-neutral-400 font-medium">Container Status</span>
							<div class="text-2xl font-bold {serviceMetrics.status === 'Healthy' ? 'text-emerald-400' : 'text-amber-400'}">
								{serviceMetrics.status}
							</div>
						</div>
					</div>
				{/if}
			</main>
		{/if}
	{:else}
		<div class="flex-1 flex items-center justify-center text-neutral-500 font-mono text-xs">
			Loading project details...
		</div>
	{/if}
</div>
