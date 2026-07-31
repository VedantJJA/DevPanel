<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { Container, Volume, SystemStats, BlueprintItem, DeleteTarget, ErrorModalState, LogStreamState, ProjectDetail } from '$lib/types';
	
	import Sidebar from '$lib/components/Sidebar.svelte';
	import DashboardView from '$lib/components/views/DashboardView.svelte';
	import ContainersView from '$lib/components/views/ContainersView.svelte';
	import BlueprintsView from '$lib/components/views/BlueprintsView.svelte';
	import SettingsView from '$lib/components/views/SettingsView.svelte';
	import ConfirmDeleteModal from '$lib/components/modals/ConfirmDeleteModal.svelte';
	import ErrorAlertModal from '$lib/components/modals/ErrorAlertModal.svelte';
	import LogStreamModal from '$lib/components/modals/LogStreamModal.svelte';
	import DeploymentLogsModal from '$lib/components/modals/DeploymentLogsModal.svelte';
	import Terminal from '$lib/components/Terminal.svelte';
	import EnvVarEditor from '$lib/components/EnvVarEditor.svelte';
	import DropdownMenu from '$lib/components/DropdownMenu.svelte';
	import { getProject, updateService, triggerDeploy, restartService } from '$lib/api';

	// Navigation & View State
	let appView = $state('dashboard'); // 'dashboard', 'containers', 'blueprints', 'workspaces', 'settings', 'detail'
	let selectedProject = $state<ProjectDetail | null>(null);
	let selectedServiceIdx = $state(0);
	let serviceTab = $state('events'); // 'events', 'logs', 'env', 'domains', 'metrics', 'settings'
	let mobileMenuOpen = $state(false);

	let loading = $state(true);
	let actionLoading = $state<string | null>(null);
	let showDeployLogsFor = $state<string | null>(null);
	let errorMessage = $state<string | null>(null);
	let pingMs = $state<number | null>(null);
	let autoRefreshRateSec = $state(5);
	let githubUsername = $state('');
	let githubToken = $state('');
	let showEnvValues = $state(false);

	let containers = $state<Container[]>([]);
	let volumes = $state<Volume[]>([]);
	let theme = $state<'light' | 'dark'>('light');
	let blueprints = $state<BlueprintItem[]>([]);
	let systemStats = $state<SystemStats>({
		totalContainers: 0,
		activeContainers: 0,
		stoppedContainers: 0,
		totalMemMb: 0,
		usedMemMb: 0,
		memPercent: 0,
		cpus: 1
	});

	// Modals State
	let selectedContainerLogs = $state<LogStreamState | null>(null);
	let deleteTarget = $state<DeleteTarget | null>(null);
	let forceDelete = $state(false);
	let errorModal = $state<ErrorModalState | null>(null);

	let logSocket: WebSocket | null = null;
	let pollInterval: ReturnType<typeof setInterval> | null = null;
	let pingInterval: ReturnType<typeof setInterval> | null = null;

	async function measurePing() {
		const start = performance.now();
		try {
			const res = await fetch('/healthz', { cache: 'no-store' });
			if (res.ok) {
				pingMs = Math.round(performance.now() - start);
			}
		} catch (e) {
			pingMs = null;
		}
	}

	async function fetchData() {
		loading = true;
		errorMessage = null;
		try {
			const [containersRes, statsRes, volumesRes, blueprintsRes] = await Promise.all([
				fetch('/api/containers'),
				fetch('/api/system/stats'),
				fetch('/api/volumes'),
				fetch('/api/blueprints')
			]);

			if (containersRes.ok) {
				const data = await containersRes.json();
				containers = data.containers || [];
			}

			if (statsRes.ok) {
				const stats = await statsRes.json();
				systemStats = {
					totalContainers: stats.totalContainers ?? containers.length,
					activeContainers: stats.activeContainers ?? containers.filter(c => c.status === 'running').length,
					stoppedContainers: stats.stoppedContainers ?? containers.filter(c => c.status !== 'running').length,
					totalMemMb: stats.totalMemMb || 0,
					usedMemMb: stats.usedMemMb || 0,
					memPercent: stats.memPercent || 0,
					cpuPercent: stats.cpuPercent ?? 0,
					cpus: stats.cpus || 1,
					os: stats.os,
					arch: stats.arch
				};
			}

			if (volumesRes.ok) {
				const vData = await volumesRes.json();
				volumes = vData.volumes || [];
			}

			if (blueprintsRes.ok) {
				const bpData = await blueprintsRes.json();
				blueprints = bpData.blueprints || [];
			}
		} catch (err: any) {
			console.error('Error fetching live telemetry:', err.message);
			errorMessage = `Unable to connect to Docker runtime API: ${err.message}`;
		} finally {
			loading = false;
		}
	}

	async function handleSelectService(container: Container) {
		const projName = container.name.replace(/^devpnl-/, '').replace(/-[^-]+$/, '');
		try {
			const data = await getProject(projName);
			selectedProject = data;
			selectedServiceIdx = 0;
			serviceTab = 'events';
			appView = 'detail';
		} catch (e) {
			openLogStream(container);
		}
	}

	function handleBackToDashboard() {
		selectedProject = null;
		appView = 'dashboard';
		mobileMenuOpen = false;
	}

	function navigateTo(view: string) {
		appView = view;
		selectedProject = null;
		mobileMenuOpen = false;
	}

	async function toggleContainerStatus(container: Container) {
		actionLoading = container.id;
		const action = container.status === 'running' ? 'stop' : 'start';
		try {
			const res = await fetch(`/api/containers/${action}?id=${container.id}`, { method: 'POST' });
			if (!res.ok) throw new Error(await res.text());
			await fetchData();
		} catch (e: any) {
			openErrorPopup(`Container ${action} Failed`, `Unable to ${action} container '${container.name}'.`, e.message);
		} finally {
			actionLoading = null;
		}
	}

	function promptDeleteContainer(container: Container) {
		deleteTarget = { type: 'container', idOrName: container.id, label: container.name };
		forceDelete = false;
	}

	function promptDeleteBlueprint(bp: BlueprintItem) {
		deleteTarget = { type: 'blueprint', idOrName: bp.id, label: bp.name };
		forceDelete = false;
	}

	async function handleDeployBlueprint(bp: BlueprintItem) {
		actionLoading = bp.id;
		try {
			const res = await fetch('/api/blueprints/deploy', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ repo_url: bp.repo_url, app_name: bp.name })
			});
			const data = await res.json();
			if (!res.ok || data.error) {
				openErrorPopup('Deployment Error', data.error || 'Failed to deploy application blueprint.', data.details);
			} else {
				await fetchData();
			}
		} catch (err: any) {
			openErrorPopup('Network Request Error', `Failed to send deployment command: ${err.message}`);
		} finally {
			actionLoading = null;
		}
	}

	async function confirmDelete() {
		if (!deleteTarget) return;
		const target = deleteTarget;
		deleteTarget = null;
		actionLoading = target.idOrName;
		try {
			if (target.type === 'container') {
				const res = await fetch(`/api/containers/delete?id=${target.idOrName}&force=${forceDelete}`, { method: 'DELETE' });
				const data = await res.json();
				if (!res.ok || data.error) {
					openErrorPopup('Container Deletion Error', data.error || 'Failed to remove container.', data.details);
				} else {
					await fetchData();
				}
			} else if (target.type === 'blueprint') {
				const res = await fetch(`/api/blueprints/delete?id=${encodeURIComponent(target.idOrName)}`, { method: 'DELETE' });
				const data = await res.json();
				if (!res.ok || data.error) {
					openErrorPopup('Blueprint Deletion Error', data.error || 'Failed to remove blueprint.');
				} else {
					await fetchData();
				}
			}
		} catch (err: any) {
			openErrorPopup('Network Request Error', `Failed to execute delete command: ${err.message}`);
		} finally {
			actionLoading = null;
		}
	}

	function openErrorPopup(title: string, message: string, details?: string) {
		errorModal = { title, message, details };
	}

	function openLogStream(container: Container) {
		closeLogStream();
		selectedContainerLogs = { id: container.id, name: container.name, logs: [`[SYS] Connecting WebSocket log stream for ${container.name}...`] };
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${protocol}//${window.location.host}/ws/logs?id=${container.id}&tail=100`;

		try {
			logSocket = new WebSocket(wsUrl);
			logSocket.onopen = () => selectedContainerLogs?.logs.push(`[SYS] WebSocket connected`);
			logSocket.onmessage = (e) => {
				try {
					const msg = JSON.parse(e.data);
					if (msg.type === 'log' && Array.isArray(msg.data)) {
						const newLines = msg.data.map((l: any) => `${l.timestamp ? `[${l.timestamp}] ` : ''}${l.line}`);
						if (selectedContainerLogs) selectedContainerLogs.logs = [...selectedContainerLogs.logs, ...newLines];
					}
				} catch (err) {
					if (selectedContainerLogs) selectedContainerLogs.logs.push(e.data);
				}
			};
			logSocket.onclose = () => selectedContainerLogs?.logs.push(`[SYS] Log stream closed`);
		} catch (err: any) {
			selectedContainerLogs?.logs.push(`[ERR] WebSocket error: ${err.message}`);
		}
	}

	function closeLogStream() {
		if (logSocket) {
			logSocket.close();
			logSocket = null;
		}
		selectedContainerLogs = null;
	}

	function updateRefreshRate(rate: number) {
		autoRefreshRateSec = rate;
		if (pollInterval) clearInterval(pollInterval);
		pollInterval = setInterval(fetchData, autoRefreshRateSec * 1000);
	}

	async function handlePruneSystem() {
		actionLoading = 'prune';
		try {
			const res = await fetch('/api/system/prune', { method: 'POST' });
			if (!res.ok) throw new Error(await res.text());
			await fetchData();
		} catch (e: any) {
			openErrorPopup('Prune Error', `System prune failed: ${e.message}`);
		} finally {
			actionLoading = null;
		}
	}

	function setTheme(nextTheme: 'light' | 'dark') {
		theme = nextTheme;
		if (typeof document !== 'undefined') {
			document.documentElement.classList.toggle('dark', nextTheme === 'dark');
			localStorage.setItem('devpnl_theme', nextTheme);
		}
	}

	onMount(async () => {
		const savedTheme = localStorage.getItem('devpnl_theme');
		setTheme(savedTheme === 'dark' ? 'dark' : 'light');

		try {
			const res = await fetch('/api/settings');
			if (res.ok) {
				const settings = await res.json();
				if (settings.github_username) githubUsername = settings.github_username;
				if (settings.github_token) githubToken = settings.github_token;
			}
		} catch (e) {
			console.error('Failed to fetch settings:', e);
		}
		
		if (typeof window !== 'undefined' && !githubUsername) {
			githubUsername = localStorage.getItem('devpnl_gh_username') || '';
		}
		fetchData();
		measurePing();
		pollInterval = setInterval(fetchData, autoRefreshRateSec * 1000);
		pingInterval = setInterval(measurePing, 3000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
		if (pingInterval) clearInterval(pingInterval);
		closeLogStream();
	});
</script>

<div class="min-h-screen bg-gray-50 text-gray-900 font-sans flex h-screen overflow-hidden selection:bg-blue-200 selection:text-blue-900">
	<!-- Desktop Sidebar -->
	<Sidebar
		{appView}
		{selectedProject}
		{selectedServiceIdx}
		{systemStats}
		{pingMs}
		onNavigate={navigateTo}
		onSelectServiceTab={(tab) => (serviceTab = tab)}
		activeServiceTab={serviceTab}
		onBackToDashboard={handleBackToDashboard}
	/>

	<!-- Main Content Area -->
	<main class="flex-1 flex flex-col min-w-0 bg-gray-50 overflow-hidden relative">
		<!-- Mobile Header Bar -->
		<div class="md:hidden h-16 border-b border-gray-200 bg-white flex items-center justify-between px-4 shrink-0 z-10">
			<div class="flex items-center gap-3">
				<button type="button" aria-label="Toggle mobile menu" class="text-gray-600 p-1 hover:bg-gray-100 rounded-lg" onclick={() => (mobileMenuOpen = !mobileMenuOpen)}>
					<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/></svg>
				</button>
				<div class="flex items-center gap-2 text-gray-900 font-bold text-lg tracking-tight">
					<div class="w-7 h-7 bg-blue-600 rounded-md flex items-center justify-center shadow-sm text-white">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/></svg>
					</div>
					<span>DevPanel</span>
				</div>
			</div>
		</div>

		<!-- Dynamic View Area -->
		<div class="flex-1 overflow-y-auto">
			{#if !selectedProject && appView === 'dashboard'}
				<DashboardView
					{containers}
					{systemStats}
					onSelectService={handleSelectService}
					onNewService={() => window.location.assign('/new')}
					onToggleStatus={toggleContainerStatus}
					onOpenLogs={openLogStream}
					onPromptDelete={promptDeleteContainer}
				/>
			{:else if !selectedProject && appView === 'containers'}
				<ContainersView
					{containers}
					{loading}
					{actionLoading}
					onToggleStatus={toggleContainerStatus}
					onOpenLogs={openLogStream}
					onPromptDelete={promptDeleteContainer}
				/>
			{:else if !selectedProject && appView === 'blueprints'}
				<BlueprintsView
					{blueprints}
					{loading}
					{actionLoading}
					onDeployBlueprint={handleDeployBlueprint}
					onPromptDeleteBlueprint={promptDeleteBlueprint}
					onCreateBlueprint={() => window.location.assign('/new')}
				/>
			{:else if !selectedProject && appView === 'workspaces'}
				<div class="p-10 flex flex-col items-center justify-center h-full text-center">
					<svg class="w-12 h-12 text-gray-300 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
					<h2 class="text-xl font-semibold text-gray-900">Workspaces</h2>
					<p class="text-gray-500 mt-2">Manage your workspaces, team access, and billing here.</p>
				</div>
			{:else if !selectedProject && appView === 'settings'}
				<SettingsView
					{autoRefreshRateSec}
					{githubUsername}
					{githubToken}
					{actionLoading}
					{theme}
					onSetTheme={setTheme}
					onSetAutoRefresh={updateRefreshRate}
					onSetGithubUsername={(username: string) => {
						githubUsername = username;
						if (typeof window !== 'undefined') {
							localStorage.setItem('devpnl_gh_username', username);
						}
						fetch('/api/settings', {
							method: 'POST',
							headers: { 'Content-Type': 'application/json' },
							body: JSON.stringify({ github_username: username })
						});
					}}
					onSetGithubToken={(token: string) => {
						githubToken = token;
						fetch('/api/settings', {
							method: 'POST',
							headers: { 'Content-Type': 'application/json' },
							body: JSON.stringify({ github_token: token })
						});
					}}
					onPruneSystem={handlePruneSystem}
				/>
			{:else if selectedProject && selectedProject.services[selectedServiceIdx]}
				{@const currentSvc = selectedProject!.services[selectedServiceIdx]}
				{@const currentSvcContainer = containers.find(c => c.name === `devpnl-${selectedProject!.blueprint.name.toLowerCase()}-${currentSvc.name.toLowerCase()}`)}
				{@const svcStatus = currentSvcContainer ? currentSvcContainer.status : 'stopped'}
				
				<div class="flex flex-col h-full bg-gray-50">
					<!-- Detail Header -->
					<header class="border-b border-gray-200 bg-white pt-6 pb-6 px-6 md:px-10 z-10 shadow-sm">
						<div class="flex flex-col md:flex-row md:items-start justify-between gap-4">
							<div class="flex items-start gap-4">
								<div class="p-3 bg-blue-50 border border-blue-100 rounded-xl text-blue-600 shadow-sm relative">
									<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/></svg>
									<!-- Status Indicator -->
									<div class="absolute -top-1.5 -right-1.5 w-4 h-4 rounded-full border-2 border-white {svcStatus === 'running' ? 'bg-emerald-500' : svcStatus === 'restarting' ? 'bg-yellow-500' : 'bg-red-500'}" title={`Status: ${svcStatus}`}></div>
								</div>
								<div>
									<div class="flex items-center gap-2 mb-1">
										<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>
										<span class="text-sm font-medium text-gray-500">{selectedProject.blueprint.name}</span>
									</div>
									<h1 class="text-2xl font-bold text-gray-900 mb-2 flex items-center gap-3 tracking-tight">
										<span>{currentSvc.name}</span>
										<span class="px-2.5 py-0.5 rounded-full text-xs font-medium border bg-green-100 text-green-700 border-green-200 uppercase">
											{currentSvc.type}
										</span>
										<span class="px-2.5 py-0.5 rounded-full text-xs font-medium border {svcStatus === 'running' ? 'bg-emerald-50 text-emerald-700 border-emerald-200' : svcStatus === 'restarting' ? 'bg-yellow-50 text-yellow-700 border-yellow-200' : 'bg-red-50 text-red-700 border-red-200'} capitalize">
											{svcStatus}
										</span>
									</h1>
									{#if currentSvc.type === 'static' || currentSvc.type === 'web'}
										<a
											href={`/app/${selectedProject.blueprint.name.toLowerCase()}`}
											target="_blank"
											rel="noreferrer"
											class="text-blue-600 hover:text-blue-700 text-sm flex items-center gap-1.5 transition-colors font-medium"
										>
											<span>{typeof window !== 'undefined' ? window.location.origin : ''}/app/{selectedProject.blueprint.name.toLowerCase()}</span>
											<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/></svg>
										</a>
									{/if}
								</div>
							</div>

							<div class="flex items-center gap-3">
								<button
									type="button"
									onclick={() => {
										showDeployLogsFor = selectedProject!.blueprint.id || selectedProject!.blueprint.name;
									}}
									class="bg-blue-50 hover:bg-blue-100 text-blue-700 px-4 py-2 rounded-lg text-sm font-medium transition-colors border border-blue-200 shadow-sm flex items-center gap-2"
								>
									<svg class="w-4 h-4 animate-pulse" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
									View Deployment Logs
								</button>
								<button
									type="button"
									onclick={async () => {
										const pid = selectedProject!.blueprint.id || selectedProject!.blueprint.name;
										showDeployLogsFor = pid;
										await triggerDeploy(pid);
									}}
									class="bg-white hover:bg-gray-50 text-gray-700 px-4 py-2 rounded-lg text-sm font-medium transition-colors border border-gray-300 shadow-sm"
								>
									Manual Deploy
								</button>
								<button
									type="button"
									onclick={() => restartService(selectedProject!.blueprint.id || selectedProject!.blueprint.name, currentSvc.name)}
									class="bg-white hover:bg-gray-50 text-gray-700 px-4 py-2 rounded-lg text-sm font-medium transition-colors border border-gray-300 shadow-sm"
								>
									Restart Service
								</button>
							</div>
						</div>
					</header>

					<!-- Tab Content -->
					<div class="flex-1 overflow-y-auto p-6 md:p-10">
						<div class="max-w-5xl mx-auto w-full">
							{#if serviceTab === 'events'}
								<div class="space-y-6">
									<h3 class="text-lg font-medium text-gray-900 mb-2">Service Configuration</h3>
									
									<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
										<div class="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
											<div class="flex items-center gap-2 mb-3 text-gray-700">
												<svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"/></svg>
												<h4 class="font-medium">Build Command</h4>
											</div>
											<div class="bg-gray-900 rounded-lg p-4 relative group">
												<code class="text-emerald-400 font-mono text-sm break-all">
													{currentSvc.build_command || 'None (Auto-detected)'}
												</code>
											</div>
											<p class="text-xs text-gray-500 mt-3">Executed during the image build phase.</p>
										</div>

										<div class="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
											<div class="flex items-center gap-2 mb-3 text-gray-700">
												<svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
												<h4 class="font-medium">Start Command</h4>
											</div>
											<div class="bg-gray-900 rounded-lg p-4 relative group">
												<code class="text-blue-400 font-mono text-sm break-all">
													{currentSvc.start_command || 'None (Container default)'}
												</code>
											</div>
											<p class="text-xs text-gray-500 mt-3">Executed when the container starts.</p>
										</div>
									</div>
								</div>
							{:else if serviceTab === 'logs'}
								<div class="h-[550px]">
									<Terminal projectId={selectedProject.blueprint.id || selectedProject.blueprint.name} serviceFilter={currentSvc.name} title={`Live Terminal Stream: ${currentSvc.name}`} />
								</div>
							{:else if serviceTab === 'env'}
								<div class="space-y-6">
									<div class="flex justify-between items-center">
										<div>
											<h3 class="text-lg font-medium text-gray-900">Environment Variables</h3>
											<p class="text-sm text-gray-500 mt-1">Manage configuration for your service.</p>
										</div>
										<button
											type="button"
											onclick={() => (showEnvValues = !showEnvValues)}
											class="bg-white hover:bg-gray-50 text-gray-700 border border-gray-300 px-3 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm"
										>
											{showEnvValues ? 'Hide Values' : 'Reveal Values'}
										</button>
									</div>
									<div class="bg-white border border-gray-200 rounded-xl p-6 shadow-sm">
										<EnvVarEditor bind:envVars={currentSvc.env_vars} />
										<div class="mt-4 flex justify-end">
											<button
												type="button"
												onclick={() => updateService(selectedProject!.blueprint.id || selectedProject!.blueprint.name, currentSvc.name, currentSvc)}
												class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors shadow-sm"
											>
												Save Environment Variables
											</button>
										</div>
									</div>
								</div>
							{:else if serviceTab === 'domains'}
								<div class="space-y-6 max-w-4xl">
									<div class="flex justify-between items-center">
										<div>
											<h3 class="text-lg font-medium text-gray-900">Custom Domains</h3>
											<p class="text-sm text-gray-500 mt-1">Manage custom domains and SSL routing.</p>
										</div>
									</div>
									<div class="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
										<div class="flex items-center justify-between">
											<div>
												<h4 class="font-medium text-gray-900">{typeof window !== 'undefined' ? window.location.origin : ''}/app/{selectedProject.blueprint.name.toLowerCase()}</h4>
												<span class="inline-block mt-1 text-xs text-gray-500 bg-gray-100 px-2 py-0.5 rounded-full border border-gray-200">Default Path URL</span>
											</div>
											<a href={`/app/${selectedProject.blueprint.name.toLowerCase()}`} target="_blank" rel="noreferrer" class="text-blue-600 hover:text-blue-700 text-sm font-medium flex items-center gap-1">
												Visit →
											</a>
										</div>
									</div>
								</div>
							{:else if serviceTab === 'metrics'}
								<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
									{#each ['CPU Usage', 'Memory Usage', 'Bandwidth', 'Requests'] as metric}
										<div class="bg-white border border-gray-200 rounded-xl p-5 h-64 flex flex-col shadow-sm">
											<h4 class="text-sm font-medium text-gray-700 mb-4">{metric}</h4>
											<div class="flex-1 flex items-center justify-center text-gray-400 border-2 border-dashed border-gray-100 rounded-lg bg-gray-50">
												<span class="text-sm font-medium">Telemetry Online</span>
											</div>
										</div>
									{/each}
								</div>
							{:else if serviceTab === 'settings'}
								<div class="space-y-8 max-w-3xl">
									<div class="bg-white border border-gray-200 rounded-xl p-6 space-y-5 shadow-sm">
										<div>
											<label for="svcSettingName" class="block text-sm font-medium text-gray-700 mb-1.5">Service Name</label>
											<input id="svcSettingName" type="text" bind:value={currentSvc.name} class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm" />
										</div>
										<div>
											<label for="svcSettingBuild" class="block text-sm font-medium text-gray-700 mb-1.5">Build Command</label>
											<input id="svcSettingBuild" type="text" bind:value={currentSvc.build_command} class="w-full font-mono bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm" />
										</div>
										<div>
											<label for="svcSettingStart" class="block text-sm font-medium text-gray-700 mb-1.5">Start Command</label>
											<input id="svcSettingStart" type="text" bind:value={currentSvc.start_command} class="w-full font-mono bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm" />
										</div>
										<button
											type="button"
											onclick={() => updateService(selectedProject!.blueprint.id || selectedProject!.blueprint.name, currentSvc.name, currentSvc)}
											class="bg-blue-600 hover:bg-blue-700 text-white px-5 py-2.5 rounded-lg text-sm font-medium transition-colors shadow-sm"
										>
											Save Settings
										</button>
									</div>
								</div>
							{/if}
						</div>
					</div>
				</div>
			{/if}
		</div>
	</main>

	<!-- Modals -->
	{#if deleteTarget}
		<ConfirmDeleteModal {deleteTarget} {forceDelete} onForceChange={(v) => (forceDelete = v)} onConfirm={confirmDelete} onCancel={() => (deleteTarget = null)} />
	{/if}
	{#if errorModal}
		<ErrorAlertModal {errorModal} onClose={() => (errorModal = null)} />
	{/if}
	{#if selectedContainerLogs}
		<LogStreamModal {selectedContainerLogs} onClose={closeLogStream} />
	{/if}
	{#if showDeployLogsFor}
		<DeploymentLogsModal projectId={showDeployLogsFor} onClose={() => (showDeployLogsFor = null)} />
	{/if}
</div>
