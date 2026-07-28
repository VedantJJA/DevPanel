<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';

	interface LogItem {
		timestamp: string;
		stage: string;
		service: string;
		message: string;
		level: string;
	}

	let projectId = $derived(page.params.id || 'my-monorepo-startup');

	// Tabs State: 'logs' | 'settings' | 'environment' | 'metrics'
	let activeTab = $state<'logs' | 'settings' | 'environment' | 'metrics'>('logs');

	// Logs Terminal State
	let logs = $state<LogItem[]>([]);
	let sseConnected = $state(false);
	let terminalContainer: HTMLDivElement | null = $state(null);

	// Editable Settings State
	let envVars = $state<Record<string, string>>({
		DB_HOST: 'database',
		DB_PORT: '5432',
		PORT: '8080',
		NODE_ENV: 'production'
	});

	let newEnvKey = $state('');
	let newEnvValue = $state('');
	let buildCommand = $state('npm ci && npm run build');
	let runCommand = $state('./server');
	let isSaving = $state(false);
	let saveMessage = $state<string | null>(null);

	// Live Metrics State
	let metricsData = $state({
		cpuPercent: 0,
		memoryMb: 0,
		status: 'Healthy',
		runningContainers: 0,
		totalContainers: 0
	});

	let eventSource: EventSource | null = null;
	let metricsInterval: any = null;

	// Auto-scroll terminal container to bottom on new log arrival
	function scrollToBottom() {
		if (terminalContainer) {
			terminalContainer.scrollTop = terminalContainer.scrollHeight;
		}
	}

	// Connect to Server-Sent Events (SSE) Stream
	function initSSEStream() {
		if (eventSource) eventSource.close();

		const sseUrl = `/api/deployments/${projectId}/logs/sse`;
		eventSource = new EventSource(sseUrl);

		eventSource.onopen = () => {
			sseConnected = true;
		};

		eventSource.addEventListener('connected', () => {
			sseConnected = true;
		});

		eventSource.onmessage = (event) => {
			try {
				const logData: LogItem = JSON.parse(event.data);
				logs = [...logs, logData];
				setTimeout(scrollToBottom, 50);
			} catch (e) {
				logs = [
					...logs,
					{
						timestamp: new Date().toISOString(),
						stage: 'system',
						service: 'console',
						message: event.data,
						level: 'info'
					}
				];
				setTimeout(scrollToBottom, 50);
			}
		};

		eventSource.onerror = () => {
			sseConnected = false;
		};
	}

	async function fetchLiveMetrics() {
		try {
			const res = await fetch('/api/containers');
			if (!res.ok) return;
			const data = await res.json();
			const list = data.containers || [];

			// Filter containers belonging to this project (e.g. devpnl-projectid-*)
			const projectContainers = list.filter((c: any) =>
				c.name.toLowerCase().includes(projectId.toLowerCase()) ||
				c.name.toLowerCase().includes('devpnl-' + projectId.toLowerCase())
			);

			if (projectContainers.length > 0) {
				let totalCpu = 0;
				let totalMem = 0;
				let runningCount = 0;
				projectContainers.forEach((c: any) => {
					totalCpu += c.cpuPercent || 0;
					totalMem += c.memoryMb || 0;
					if (c.status === 'running') runningCount++;
				});

				metricsData = {
					cpuPercent: Math.round(totalCpu * 10) / 10,
					memoryMb: Math.round(totalMem * 10) / 10,
					status: runningCount === projectContainers.length ? 'Healthy' : 'Degraded',
					runningContainers: runningCount,
					totalContainers: projectContainers.length
				};
			} else if (list.length > 0) {
				// Fallback to average across all live containers
				let totalCpu = 0;
				let totalMem = 0;
				let runningCount = 0;
				list.forEach((c: any) => {
					totalCpu += c.cpuPercent || 0;
					totalMem += c.memoryMb || 0;
					if (c.status === 'running') runningCount++;
				});

				metricsData = {
					cpuPercent: Math.round(totalCpu * 10) / 10,
					memoryMb: Math.round(totalMem * 10) / 10,
					status: runningCount > 0 ? 'Healthy' : 'Stopped',
					runningContainers: runningCount,
					totalContainers: list.length
				};
			}
		} catch (e) {
			console.error('Failed to fetch live metrics:', e);
		}
	}

	function addEnvVar() {
		if (newEnvKey.trim()) {
			envVars = { ...envVars, [newEnvKey.trim().toUpperCase()]: newEnvValue.trim() };
			newEnvKey = '';
			newEnvValue = '';
		}
	}

	function removeEnvVar(key: string) {
		const updated = { ...envVars };
		delete updated[key];
		envVars = updated;
	}

	async function handleSaveSettings() {
		isSaving = true;
		saveMessage = null;
		try {
			const res = await fetch('/api/deployments/trigger', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					project: projectId,
					repo_url: 'https://github.com/username/my-monorepo.git',
					blueprint: {
						version: '1.0',
						project: projectId,
						services: {
							backend: {
								type: 'web',
								build: { command: buildCommand },
								deploy: { command: runCommand, env: envVars }
							}
						}
					}
				})
			});

			if (res.ok) {
				saveMessage = 'Settings saved! Triggered redeployment pipeline.';
				activeTab = 'logs';
			} else {
				throw new Error(await res.text());
			}
		} catch (e: any) {
			saveMessage = `Error saving settings: ${e.message}`;
		} finally {
			isSaving = false;
		}
	}

	onMount(() => {
		initSSEStream();
		fetchLiveMetrics();
		metricsInterval = setInterval(fetchLiveMetrics, 3000);
	});

	onDestroy(() => {
		if (eventSource) {
			eventSource.close();
		}
		if (metricsInterval) {
			clearInterval(metricsInterval);
		}
	});
</script>

<div class="flex h-screen bg-neutral-950 text-neutral-100 font-sans antialiased overflow-hidden">
	<!-- Render.com-style Left Sidebar Navigation -->
	<aside class="w-64 border-r border-neutral-800 bg-neutral-900/70 backdrop-blur-md flex flex-col justify-between p-4 shrink-0">
		<div class="space-y-6">
			<!-- App Header -->
			<div class="px-3 py-2 border-b border-neutral-800/80 pb-4">
				<div class="flex items-center justify-between">
					<h2 class="font-bold text-base text-neutral-100 truncate font-mono" title={projectId}>{projectId}</h2>
					<span class="px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">Active</span>
				</div>
				<p class="text-xs text-neutral-400 mt-1 font-mono">Render-style Live Console</p>
			</div>

			<!-- Navigation Tabs -->
			<nav class="space-y-1">
				<button
					onclick={() => (activeTab = 'logs')}
					class="w-full flex items-center justify-between px-3.5 py-2.5 rounded-xl text-sm font-medium transition-all {activeTab === 'logs'
						? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
						: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
				>
					<div class="flex items-center gap-3">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
						<span>Logs Stream</span>
					</div>
					<span class="w-2 h-2 rounded-full {sseConnected ? 'bg-emerald-400 animate-pulse' : 'bg-amber-400'}"></span>
				</button>

				<button
					onclick={() => (activeTab = 'environment')}
					class="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-sm font-medium transition-all {activeTab === 'environment'
						? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
						: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"/></svg>
					<span>Environment</span>
				</button>

				<button
					onclick={() => (activeTab = 'settings')}
					class="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-sm font-medium transition-all {activeTab === 'settings'
						? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
						: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>
					<span>Settings</span>
				</button>

				<button
					onclick={() => (activeTab = 'metrics')}
					class="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-sm font-medium transition-all {activeTab === 'metrics'
						? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
						: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/></svg>
					<span>Metrics</span>
				</button>
			</nav>
		</div>

		<div class="pt-4 border-t border-neutral-800/80">
			<a href="/" class="text-xs font-medium text-neutral-400 hover:text-emerald-400 transition-colors flex items-center gap-2">
				← Back to Dashboard
			</a>
		</div>
	</aside>

	<!-- Main Console Workspace -->
	<main class="flex-1 flex flex-col min-w-0 bg-neutral-950 p-6 md:p-8 overflow-y-auto">
		<!-- Top Action Header -->
		<div class="flex items-center justify-between border-b border-neutral-800/80 pb-4 mb-6">
			<div>
				<h1 class="text-2xl font-bold text-neutral-100 font-mono flex items-center gap-3">
					<span>{projectId}</span>
					<span class="text-xs px-3 py-1 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-sans font-medium">Deployed & Running</span>
				</h1>
				<p class="text-xs text-neutral-400 font-mono mt-1">Hosted URL: <a href="/app/{projectId}" target="_blank" class="text-emerald-400 hover:underline">{typeof window !== 'undefined' ? window.location.origin : ''}/app/{projectId} ↗</a></p>
			</div>

			<div class="flex items-center gap-3">
				<button
					onclick={handleSaveSettings}
					disabled={isSaving}
					class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold transition-all shadow-md shadow-emerald-950/50 disabled:opacity-50"
				>
					{isSaving ? 'Redeploying...' : 'Manual Redeploy'}
				</button>
			</div>
		</div>

		<!-- Feedback Message -->
		{#if saveMessage}
			<div class="p-4 rounded-xl bg-neutral-900 border border-neutral-800 text-emerald-400 text-xs font-mono mb-6 flex items-center justify-between">
				<span>{saveMessage}</span>
				<button onclick={() => (saveMessage = null)} class="text-neutral-400 hover:text-neutral-200">✕</button>
			</div>
		{/if}

		<!-- TAB 1: Live Console SSE Terminal -->
		{#if activeTab === 'logs'}
			<div class="flex-1 flex flex-col min-h-0 bg-black rounded-2xl border border-neutral-800 shadow-2xl overflow-hidden font-mono text-xs">
				<!-- Terminal Bar -->
				<div class="px-4 py-3 bg-neutral-900 border-b border-neutral-800 flex items-center justify-between">
					<div class="flex items-center gap-2">
						<span class="w-3 h-3 rounded-full bg-rose-500/80"></span>
						<span class="w-3 h-3 rounded-full bg-amber-500/80"></span>
						<span class="w-3 h-3 rounded-full bg-emerald-500/80"></span>
						<span class="ml-2 text-neutral-400 text-xs">Deployment & Runtime Logs (SSE)</span>
					</div>

					<div class="flex items-center gap-3">
						<button onclick={() => (logs = [])} class="text-neutral-400 hover:text-neutral-200 text-xs">Clear Terminal</button>
						<span class="w-2 h-2 rounded-full {sseConnected ? 'bg-emerald-400' : 'bg-rose-400'}"></span>
					</div>
				</div>

				<!-- Terminal Log Viewport with Auto-Scroll -->
				<div bind:this={terminalContainer} class="flex-1 p-4 overflow-y-auto space-y-1.5 scroll-smooth">
					{#if logs.length === 0}
						<div class="text-neutral-500 italic py-4">Waiting for deployment log stream...</div>
					{/if}

					{#each logs as logItem, i (i)}
						<div class="flex items-start gap-3 leading-relaxed">
							<span class="text-neutral-600 text-[11px] shrink-0">{logItem.timestamp.slice(11, 19)}</span>
							<span class="px-1.5 py-0.5 rounded text-[10px] uppercase font-bold shrink-0 {logItem.stage === 'build'
								? 'bg-sky-950 text-sky-400 border border-sky-800'
								: logItem.stage === 'deploy'
								? 'bg-purple-950 text-purple-400 border border-purple-800'
								: 'bg-neutral-800 text-neutral-300'}">
								{logItem.stage}
							</span>
							<span class="text-neutral-400 shrink-0">[{logItem.service}]</span>
							<span class="flex-1 whitespace-pre-wrap {logItem.level === 'error'
								? 'text-rose-400 font-bold'
								: logItem.level === 'success'
								? 'text-emerald-400 font-bold'
								: 'text-neutral-200'}">
								{logItem.message}
							</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- TAB 2: Environment Variables -->
		{#if activeTab === 'environment'}
			<div class="space-y-6 max-w-4xl">
				<div class="p-6 rounded-2xl bg-neutral-900/60 border border-neutral-800 space-y-4">
					<h3 class="text-base font-bold text-neutral-100">Environment Variables</h3>
					<p class="text-xs text-neutral-400">Configure key-value variables passed to your application runtime containers.</p>

					<!-- Key Value Pairs -->
					<div class="space-y-3 pt-2">
						{#each Object.entries(envVars) as [k, v] (k)}
							<div class="flex items-center gap-3">
								<input
									type="text"
									value={k}
									readonly
									class="w-1/3 px-3 py-2 bg-neutral-950 border border-neutral-800 rounded-xl text-xs font-mono text-neutral-300"
								/>
								<input
									type="text"
									bind:value={envVars[k]}
									class="flex-1 px-3 py-2 bg-neutral-950 border border-neutral-800 rounded-xl text-xs font-mono text-emerald-400 focus:outline-none focus:border-emerald-500"
								/>
								<button
									onclick={() => removeEnvVar(k)}
									class="p-2 rounded-lg bg-neutral-800 hover:bg-rose-600/30 text-rose-400 transition-all text-xs font-bold"
								>
									✕
								</button>
							</div>
						{/each}
					</div>

					<!-- Add New Var -->
					<div class="pt-4 border-t border-neutral-800 flex items-center gap-3">
						<input
							type="text"
							bind:value={newEnvKey}
							placeholder="KEY_NAME"
							class="w-1/3 px-3 py-2 bg-neutral-950 border border-neutral-800 rounded-xl text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
						/>
						<input
							type="text"
							bind:value={newEnvValue}
							placeholder="value"
							class="flex-1 px-3 py-2 bg-neutral-950 border border-neutral-800 rounded-xl text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
						/>
						<button
							onclick={addEnvVar}
							class="px-4 py-2 rounded-xl bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-semibold"
						>
							+ Add Variable
						</button>
					</div>
				</div>

				<div class="flex justify-end">
					<button
						onclick={handleSaveSettings}
						disabled={isSaving}
						class="px-6 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition-all shadow-lg"
					>
						Save & Redeploy Application
					</button>
				</div>
			</div>
		{/if}

		<!-- TAB 3: Build & Run Settings -->
		{#if activeTab === 'settings'}
			<div class="space-y-6 max-w-4xl">
				<div class="p-6 rounded-2xl bg-neutral-900/60 border border-neutral-800 space-y-4">
					<h3 class="text-base font-bold text-neutral-100">Build & Runtime Settings</h3>

					<div class="space-y-2">
						<label for="bCmd" class="block text-xs font-semibold uppercase text-neutral-400">Build Command</label>
						<input
							id="bCmd"
							type="text"
							bind:value={buildCommand}
							class="w-full px-4 py-3 bg-neutral-950 border border-neutral-800 rounded-xl text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
						/>
					</div>

					<div class="space-y-2 pt-2">
						<label for="rCmd" class="block text-xs font-semibold uppercase text-neutral-400">Run Command</label>
						<input
							id="rCmd"
							type="text"
							bind:value={runCommand}
							class="w-full px-4 py-3 bg-neutral-950 border border-neutral-800 rounded-xl text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
						/>
					</div>
				</div>

				<div class="flex justify-end">
					<button
						onclick={handleSaveSettings}
						disabled={isSaving}
						class="px-6 py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs transition-all shadow-lg"
					>
						Save Settings
					</button>
				</div>
			</div>
		{/if}

		<!-- TAB 4: Real-time Live Metrics -->
		{#if activeTab === 'metrics'}
			<div class="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-4xl">
				<div class="p-5 rounded-2xl bg-neutral-900/60 border border-neutral-800 space-y-2">
					<span class="text-xs text-neutral-400 font-medium">CPU Utilization</span>
					<div class="text-2xl font-bold text-neutral-100 font-mono">{metricsData.cpuPercent}%</div>
					<span class="text-[11px] text-neutral-500 block">Live container CPU load</span>
				</div>

				<div class="p-5 rounded-2xl bg-neutral-900/60 border border-neutral-800 space-y-2">
					<span class="text-xs text-neutral-400 font-medium">Memory Usage</span>
					<div class="text-2xl font-bold text-neutral-100 font-mono">{metricsData.memoryMb} MB</div>
					<span class="text-[11px] text-neutral-500 block">Live cgroups memory RSS</span>
				</div>

				<div class="p-5 rounded-2xl bg-neutral-900/60 border border-neutral-800 space-y-2">
					<span class="text-xs text-neutral-400 font-medium">Container Health</span>
					<div class="text-2xl font-bold {metricsData.status === 'Healthy' ? 'text-emerald-400' : 'text-amber-400'}">
						{metricsData.status} ({metricsData.runningContainers}/{metricsData.totalContainers} Online)
					</div>
					<span class="text-[11px] text-neutral-500 block">Active container status</span>
				</div>
			</div>
		{/if}
	</main>
</div>
