<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { Container, Volume, SystemStats, DeleteTarget, ErrorModalState, LogStreamState, TabType } from '$lib/types';
	
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Header from '$lib/components/Header.svelte';
	import MetricsOverview from '$lib/components/MetricsOverview.svelte';
	import OverviewTab from '$lib/components/tabs/OverviewTab.svelte';
	import ContainersTab from '$lib/components/tabs/ContainersTab.svelte';
	import VolumesTab from '$lib/components/tabs/VolumesTab.svelte';
	import SettingsTab from '$lib/components/tabs/SettingsTab.svelte';
	import ConfirmDeleteModal from '$lib/components/modals/ConfirmDeleteModal.svelte';
	import ErrorAlertModal from '$lib/components/modals/ErrorAlertModal.svelte';
	import LogStreamModal from '$lib/components/modals/LogStreamModal.svelte';

	// Main Application State
	let activeTab = $state<TabType>('overview');
	let loading = $state(true);
	let actionLoading = $state<string | null>(null);
	let errorMessage = $state<string | null>(null);
	let pingMs = $state<number | null>(null);
	let autoRefreshRateSec = $state(5);

	let containers = $state<Container[]>([]);
	let volumes = $state<Volume[]>([]);
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

	// Measure Webpage Ping / Server Latency
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

	// Fetch live container list, volumes & system telemetry from Go API
	async function fetchData() {
		loading = true;
		errorMessage = null;
		try {
			const [containersRes, statsRes, volumesRes] = await Promise.all([
				fetch('/api/containers'),
				fetch('/api/system/stats'),
				fetch('/api/volumes')
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
		} catch (err: any) {
			console.error('Error fetching live telemetry:', err.message);
			errorMessage = `Unable to connect to Docker runtime API: ${err.message}`;
		} finally {
			loading = false;
		}
	}

	// Batch Operations
	async function handleStartAll() {
		actionLoading = 'start-all';
		try {
			const res = await fetch('/api/containers/start-all', { method: 'POST' });
			if (!res.ok) throw new Error(await res.text());
			await fetchData();
		} catch (e: any) {
			openErrorPopup('Batch Operation Error', `Failed to start all containers: ${e.message}`);
		} finally {
			actionLoading = null;
		}
	}

	async function handleStopAll() {
		actionLoading = 'stop-all';
		try {
			const res = await fetch('/api/containers/stop-all', { method: 'POST' });
			if (!res.ok) throw new Error(await res.text());
			await fetchData();
		} catch (e: any) {
			openErrorPopup('Batch Operation Error', `Failed to stop all containers: ${e.message}`);
		} finally {
			actionLoading = null;
		}
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

	// Confirm & Delete Handlers
	function promptDeleteContainer(container: Container) {
		deleteTarget = {
			type: 'container',
			idOrName: container.id,
			label: container.name
		};
		forceDelete = false;
	}

	function promptDeleteVolume(volume: Volume) {
		deleteTarget = {
			type: 'volume',
			idOrName: volume.Name,
			label: volume.Name
		};
		forceDelete = false;
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
			} else if (target.type === 'volume') {
				const res = await fetch(`/api/volumes/delete?name=${encodeURIComponent(target.idOrName)}&force=${forceDelete}`, { method: 'DELETE' });
				const data = await res.json();
				if (!res.ok || data.error) {
					openErrorPopup('Volume Deletion Error', data.error || 'Failed to remove volume.', data.details);
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

	function openErrorPopup(title: string, message: string, details?: string) {
		errorModal = { title, message, details };
	}

	// Live WebSocket Log Streaming
	function openLogStream(container: Container) {
		closeLogStream();

		selectedContainerLogs = {
			id: container.id,
			name: container.name,
			logs: [`[SYS] Connecting live WebSocket log stream for ${container.name}...`]
		};

		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${protocol}//${window.location.host}/ws/logs?id=${container.id}&tail=100`;

		try {
			logSocket = new WebSocket(wsUrl);

			logSocket.onopen = () => {
				if (selectedContainerLogs) {
					selectedContainerLogs.logs.push(`[SYS] WebSocket log stream established`);
				}
			};

			logSocket.onmessage = (event) => {
				try {
					const msg = JSON.parse(event.data);
					if (msg.type === 'log' && Array.isArray(msg.data)) {
						const newLines = msg.data.map((l: any) => `${l.timestamp ? `[${l.timestamp}] ` : ''}${l.stream ? `[${l.stream.toUpperCase()}] ` : ''}${l.line}`);
						if (selectedContainerLogs) {
							selectedContainerLogs.logs = [...selectedContainerLogs.logs, ...newLines];
						}
					} else if (msg.type === 'error') {
						if (selectedContainerLogs) {
							selectedContainerLogs.logs.push(`[ERR] ${msg.data}`);
						}
					}
				} catch (e) {
					if (selectedContainerLogs) {
						selectedContainerLogs.logs.push(event.data);
					}
				}
			};

			logSocket.onerror = () => {
				if (selectedContainerLogs) {
					selectedContainerLogs.logs.push(`[ERR] WebSocket error streaming logs`);
				}
			};

			logSocket.onclose = () => {
				if (selectedContainerLogs) {
					selectedContainerLogs.logs.push(`[SYS] Log stream closed`);
				}
			};
		} catch (err: any) {
			if (selectedContainerLogs) {
				selectedContainerLogs.logs.push(`[ERR] Failed to connect WebSocket: ${err.message}`);
			}
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

	onMount(() => {
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

<div class="flex h-screen bg-neutral-950 text-neutral-100 font-sans antialiased overflow-hidden">
	<!-- Modular Sidebar -->
	<Sidebar
		{activeTab}
		containersCount={containers.length}
		volumesCount={volumes.length}
		{systemStats}
		{pingMs}
		onTabSelect={(tab) => (activeTab = tab)}
	/>

	<!-- Main Workspace Area -->
	<main class="flex-1 flex flex-col min-w-0 overflow-y-auto bg-neutral-950">
		<!-- Modular Header -->
		<Header
			{activeTab}
			{loading}
			{actionLoading}
			{pingMs}
			onRefresh={fetchData}
			onStartAll={handleStartAll}
			onStopAll={handleStopAll}
		/>

		<!-- Content Area -->
		<div class="p-8 space-y-8 max-w-7xl w-full mx-auto">
			<!-- Inline Error Alert Banner -->
			{#if errorMessage}
				<div class="p-4 rounded-xl bg-rose-950/50 border border-rose-800/80 text-rose-300 text-sm flex items-center justify-between">
					<div class="flex items-center gap-3">
						<svg class="w-5 h-5 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
						<span>{errorMessage}</span>
					</div>
					<button onclick={() => (errorMessage = null)} class="text-rose-400 hover:text-rose-200 text-xs font-bold">Dismiss</button>
				</div>
			{/if}

			<!-- Modular Metrics Overview Cards -->
			<MetricsOverview {containers} {systemStats} />

			<!-- Active Tab View Component -->
			{#if activeTab === 'overview'}
				<OverviewTab {containers} {volumes} {systemStats} />
			{:else if activeTab === 'containers'}
				<ContainersTab
					{containers}
					{loading}
					{actionLoading}
					onToggleStatus={toggleContainerStatus}
					onOpenLogs={openLogStream}
					onPromptDelete={promptDeleteContainer}
				/>
			{:else if activeTab === 'volumes'}
				<VolumesTab
					{volumes}
					{loading}
					onPromptDelete={promptDeleteVolume}
				/>
			{:else if activeTab === 'settings'}
				<SettingsTab
					{autoRefreshRateSec}
					{actionLoading}
					onSetAutoRefresh={updateRefreshRate}
					onPruneSystem={handlePruneSystem}
				/>
			{/if}

			<!-- Delete Confirmation Modal -->
			{#if deleteTarget}
				<ConfirmDeleteModal
					{deleteTarget}
					{forceDelete}
					onForceChange={(val) => (forceDelete = val)}
					onConfirm={confirmDelete}
					onCancel={() => (deleteTarget = null)}
				/>
			{/if}

			<!-- Error Alert Modal -->
			{#if errorModal}
				<ErrorAlertModal
					{errorModal}
					onClose={() => (errorModal = null)}
				/>
			{/if}

			<!-- WebSocket Live Log Stream Drawer -->
			{#if selectedContainerLogs}
				<LogStreamModal
					{selectedContainerLogs}
					onClose={closeLogStream}
				/>
			{/if}
		</div>
	</main>
</div>
