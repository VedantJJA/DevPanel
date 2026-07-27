<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	interface Container {
		id: string;
		name: string;
		image: string;
		status: 'running' | 'stopped' | 'restarting' | 'error';
		port: string;
		cpuPercent: number;
		memoryMb: number;
		uptime: string;
	}

	interface Volume {
		Name: string;
		Driver: string;
		Mountpoint: string;
		CreatedAt?: string;
		Scope?: string;
	}

	interface SystemStats {
		totalContainers: number;
		activeContainers: number;
		stoppedContainers: number;
		totalMemMb: number;
		usedMemMb: number;
		memPercent: number;
		cpus: number;
		os?: string;
		arch?: string;
	}

	// Active Tab State: 'overview' | 'containers' | 'volumes' | 'settings'
	let activeTab = $state<'overview' | 'containers' | 'volumes' | 'settings'>('overview');
	let searchQuery = $state('');
	let statusFilter = $state('all');
	
	let loading = $state(true);
	let actionLoading = $state<string | null>(null);
	let errorMessage = $state<string | null>(null);
	let pingMs = $state<number | null>(null);

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
	let selectedContainerLogs = $state<{ id: string; name: string; logs: string[] } | null>(null);
	
	// Delete Confirmation Modal State
	let deleteTarget = $state<{ type: 'container' | 'volume'; idOrName: string; label: string } | null>(null);
	let forceDelete = $state(false);

	// Error Alert Popup Modal State
	let errorModal = $state<{ title: string; message: string; details?: string } | null>(null);

	let logSocket: WebSocket | null = null;
	let pollInterval: ReturnType<typeof setInterval> | null = null;
	let pingInterval: ReturnType<typeof setInterval> | null = null;
	let autoRefreshRateSec = $state(5);

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

	// Calculated Dynamic Metrics
	let activeCount = $derived(containers.filter(c => c.status === 'running').length);
	let totalCpuPercent = $derived(
		Math.min(100, Math.round(containers.filter(c => c.status === 'running').reduce((acc, c) => acc + (c.cpuPercent || 0), 0) * 10) / 10)
	);

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
		deleteTarget = null; // Close confirmation modal

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

	function closeErrorPopup() {
		errorModal = null;
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

	let filteredContainers = $derived(
		containers.filter((c) => {
			const matchesSearch =
				c.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				c.image.toLowerCase().includes(searchQuery.toLowerCase());
			const matchesStatus = statusFilter === 'all' || c.status === statusFilter;
			return matchesSearch && matchesStatus;
		})
	);

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
	<!-- Sidebar Navigation -->
	<aside class="w-64 border-r border-neutral-800 bg-neutral-900/60 backdrop-blur-md flex flex-col justify-between p-4 shrink-0">
		<div>
			<!-- Brand Header -->
			<div class="flex items-center gap-3 px-3 py-3 mb-6 border-b border-neutral-800/80">
				<div class="h-9 w-9 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400 font-mono font-bold text-lg shadow-sm shadow-emerald-950">
					>_
				</div>
				<div>
					<h1 class="font-bold text-base tracking-tight text-neutral-100">DevPanel</h1>
					<p class="text-xs text-neutral-400 font-mono">{systemStats.os || 'Linux OS'} {systemStats.arch ? `(${systemStats.arch})` : ''}</p>
				</div>
			</div>

			<!-- Nav Items -->
			<nav class="space-y-1">
				<button
					onclick={() => (activeTab = 'overview')}
					class="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all {activeTab === 'overview'
						? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
						: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"/></svg>
					Overview
				</button>

				<button
					onclick={() => (activeTab = 'containers')}
					class="w-full flex items-center justify-between px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all {activeTab === 'containers'
						? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
						: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
				>
					<div class="flex items-center gap-3">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
						Containers
					</div>
					<span class="text-xs px-2 py-0.5 rounded-full bg-neutral-800 text-neutral-300 font-mono">{containers.length}</span>
				</button>

				<button
					onclick={() => (activeTab = 'volumes')}
					class="w-full flex items-center justify-between px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all {activeTab === 'volumes'
						? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
						: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
				>
					<div class="flex items-center gap-3">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
						Volumes
					</div>
					<span class="text-xs px-2 py-0.5 rounded-full bg-neutral-800 text-neutral-300 font-mono">{volumes.length}</span>
				</button>

				<button
					onclick={() => (activeTab = 'settings')}
					class="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all {activeTab === 'settings'
						? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
						: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>
					Settings
				</button>
			</nav>
		</div>

		<!-- Status & Webpage Ping Footer -->
		<div class="border-t border-neutral-800/80 pt-4 px-2 space-y-2">
			<!-- Webpage Ping RTT Indicator -->
			<div class="flex items-center justify-between text-xs text-neutral-400">
				<span>Webpage Latency</span>
				<span class="inline-flex items-center gap-1.5 font-mono text-xs {pingMs !== null ? 'text-emerald-400' : 'text-rose-400'}">
					<span class="w-2 h-2 rounded-full {pingMs !== null ? 'bg-emerald-400 animate-pulse' : 'bg-rose-500'}"></span>
					{pingMs !== null ? `${pingMs} ms` : 'Offline'}
				</span>
			</div>
			<div class="flex items-center justify-between text-xs text-neutral-400">
				<span>Scale-to-Zero</span>
				<span class="inline-flex items-center gap-1.5 text-emerald-400 font-mono">Active</span>
			</div>
			<div class="flex items-center justify-between text-xs text-neutral-500 font-mono">
				<span>Systemd Socket</span>
				<span>LISTEN_FDS=1</span>
			</div>
		</div>
	</aside>

	<!-- Main Workspace Area -->
	<main class="flex-1 flex flex-col min-w-0 overflow-y-auto bg-neutral-950">
		<!-- Top Header Bar -->
		<header class="border-b border-neutral-800 bg-neutral-900/40 px-8 py-5 flex items-center justify-between gap-4 sticky top-0 backdrop-blur-md z-10">
			<div>
				<h2 class="text-xl font-bold tracking-tight text-neutral-100 capitalize">{activeTab} Dashboard</h2>
				<p class="text-xs text-neutral-400 mt-0.5">Real-time system telemetry and Docker resource controls</p>
			</div>

			<!-- Action Bar & Ping Badge -->
			<div class="flex items-center gap-3">
				<!-- Webpage Ping Badge -->
				<div class="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-neutral-900 border border-neutral-800 text-xs font-mono text-neutral-300">
					<span class="w-2 h-2 rounded-full {pingMs !== null ? 'bg-emerald-400 animate-pulse' : 'bg-rose-500'}"></span>
					<span>Ping: <strong class="text-neutral-100">{pingMs !== null ? `${pingMs} ms` : 'Disconnected'}</strong></span>
				</div>

				<button
					onclick={fetchData}
					disabled={loading}
					class="px-3.5 py-2 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-semibold flex items-center gap-2 border border-neutral-700/60 transition-all disabled:opacity-50"
				>
					<svg class="w-3.5 h-3.5 {loading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
					Refresh
				</button>

				{#if activeTab === 'containers' || activeTab === 'overview'}
					<button
						onclick={handleStartAll}
						disabled={actionLoading !== null}
						class="px-3.5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold flex items-center gap-2 transition-all shadow-sm shadow-emerald-950 disabled:opacity-50"
					>
						{#if actionLoading === 'start-all'}
							<svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
						{:else}
							<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
						{/if}
						Start All
					</button>

					<button
						onclick={handleStopAll}
						disabled={actionLoading !== null}
						class="px-3.5 py-2 rounded-lg bg-rose-600/90 hover:bg-rose-500 text-white text-xs font-semibold flex items-center gap-2 transition-all shadow-sm shadow-rose-950 disabled:opacity-50"
					>
						{#if actionLoading === 'stop-all'}
							<svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
						{:else}
							<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"/></svg>
						{/if}
						Stop All
					</button>
				{/if}
			</div>
		</header>

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

			<!-- Dynamic System Metrics Banner -->
			<section class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5">
				<!-- Active Containers Count -->
				<div class="p-5 rounded-2xl bg-neutral-900/70 border border-neutral-800 flex flex-col justify-between shadow-sm">
					<div class="flex items-center justify-between text-neutral-400 text-xs font-medium">
						<span>Active Containers</span>
						<span class="p-1.5 rounded-lg bg-emerald-500/10 text-emerald-400">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/></svg>
						</span>
					</div>
					<div class="mt-3 flex items-baseline gap-2">
						<span class="text-3xl font-bold text-neutral-100 font-mono">{activeCount}</span>
						<span class="text-xs text-neutral-400 font-mono">/ {containers.length} total</span>
					</div>
					<div class="mt-3 w-full bg-neutral-800 h-1.5 rounded-full overflow-hidden">
						<div class="bg-emerald-500 h-full transition-all duration-500" style="width: {containers.length ? (activeCount / containers.length) * 100 : 0}%"></div>
					</div>
				</div>

				<!-- Host CPU Core Count & Live Load -->
				<div class="p-5 rounded-2xl bg-neutral-900/70 border border-neutral-800 flex flex-col justify-between shadow-sm">
					<div class="flex items-center justify-between text-neutral-400 text-xs font-medium">
						<span>CPU Core Count & Load</span>
						<span class="p-1.5 rounded-lg bg-sky-500/10 text-sky-400">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M3 9h2m-2 6h2m14-6h2m-2 6h2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/></svg>
						</span>
					</div>
					<div class="mt-3 flex items-baseline gap-2">
						<span class="text-3xl font-bold text-neutral-100 font-mono">{systemStats.cpus} <span class="text-sm font-normal text-neutral-400">{systemStats.cpus === 1 ? 'Core' : 'Cores'}</span></span>
						<span class="text-xs text-sky-400 font-mono">({totalCpuPercent}% load)</span>
					</div>
					<div class="mt-3 w-full bg-neutral-800 h-1.5 rounded-full overflow-hidden">
						<div class="bg-sky-500 h-full transition-all duration-500" style="width: {totalCpuPercent}%"></div>
					</div>
				</div>

				<!-- Automatic Host RAM Usage & Total Max RAM -->
				<div class="p-5 rounded-2xl bg-neutral-900/70 border border-neutral-800 flex flex-col justify-between shadow-sm">
					<div class="flex items-center justify-between text-neutral-400 text-xs font-medium">
						<span>Memory (Used / Max)</span>
						<span class="p-1.5 rounded-lg bg-purple-500/10 text-purple-400">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
						</span>
					</div>
					<div class="mt-3 flex items-baseline gap-2">
						<span class="text-3xl font-bold text-neutral-100 font-mono">{systemStats.usedMemMb} <span class="text-sm font-normal text-neutral-400">MB</span></span>
						<span class="text-xs text-purple-300 font-mono">/ {systemStats.totalMemMb} MB</span>
					</div>
					<div class="mt-3 w-full bg-neutral-800 h-1.5 rounded-full overflow-hidden">
						<div class="bg-purple-500 h-full transition-all duration-500" style="width: {systemStats.memPercent}%"></div>
					</div>
				</div>

				<!-- Engine Status -->
				<div class="p-5 rounded-2xl bg-neutral-900/70 border border-neutral-800 flex flex-col justify-between shadow-sm">
					<div class="flex items-center justify-between text-neutral-400 text-xs font-medium">
						<span>Server Host Engine</span>
						<span class="p-1.5 rounded-lg bg-emerald-500/10 text-emerald-400">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
						</span>
					</div>
					<div class="mt-3 flex items-baseline gap-2">
						<span class="text-2xl font-bold text-emerald-400">Online</span>
					</div>
					<p class="mt-3 text-xs text-neutral-400">Scale-to-Zero & SQLite ready</p>
				</div>
			</section>

			<!-- TAB 1: OVERVIEW PAGE -->
			{#if activeTab === 'overview'}
				<div class="space-y-6">
					<h3 class="text-lg font-bold text-neutral-100">System Runtime Overview</h3>
					<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
						<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-4">
							<h4 class="font-semibold text-neutral-200 text-sm">Host System Architecture</h4>
							<div class="space-y-2 text-xs font-mono text-neutral-300">
								<div class="flex justify-between border-b border-neutral-800/60 py-2">
									<span class="text-neutral-400">Operating System:</span>
									<span>{systemStats.os || 'Linux Runtime'}</span>
								</div>
								<div class="flex justify-between border-b border-neutral-800/60 py-2">
									<span class="text-neutral-400">Architecture:</span>
									<span>{systemStats.arch || 'arm64 / amd64'}</span>
								</div>
								<div class="flex justify-between border-b border-neutral-800/60 py-2">
									<span class="text-neutral-400">CPU Cores:</span>
									<span>{systemStats.cpus} Cores</span>
								</div>
								<div class="flex justify-between py-2">
									<span class="text-neutral-400">Total Host RAM:</span>
									<span>{systemStats.totalMemMb} MB</span>
								</div>
							</div>
						</div>

						<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-4">
							<h4 class="font-semibold text-neutral-200 text-sm">Docker Engine Overview</h4>
							<div class="space-y-2 text-xs font-mono text-neutral-300">
								<div class="flex justify-between border-b border-neutral-800/60 py-2">
									<span class="text-neutral-400">Total Containers:</span>
									<span>{containers.length}</span>
								</div>
								<div class="flex justify-between border-b border-neutral-800/60 py-2">
									<span class="text-neutral-400">Running Containers:</span>
									<span class="text-emerald-400">{activeCount}</span>
								</div>
								<div class="flex justify-between border-b border-neutral-800/60 py-2">
									<span class="text-neutral-400">Total Volumes:</span>
									<span>{volumes.length}</span>
								</div>
								<div class="flex justify-between py-2">
									<span class="text-neutral-400">Scale-to-Zero Idle Timeout:</span>
									<span>5 Minutes</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			{/if}

			<!-- TAB 2: CONTAINERS PAGE -->
			{#if activeTab === 'containers' || activeTab === 'overview'}
				<section class="space-y-4">
					<div class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-4">
						<h3 class="text-lg font-bold text-neutral-100">Containers</h3>
						<div class="flex items-center gap-3">
							<div class="relative flex-1 max-w-xs">
								<svg class="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-neutral-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>
								<input
									type="text"
									bind:value={searchQuery}
									placeholder="Filter container name..."
									class="w-full pl-10 pr-4 py-2 bg-neutral-900 border border-neutral-800 rounded-xl text-xs text-neutral-200 focus:outline-none focus:border-emerald-500/50"
								/>
							</div>

							<div class="bg-neutral-900 border border-neutral-800 p-1 rounded-xl flex items-center gap-1">
								<button
									onclick={() => (statusFilter = 'all')}
									class="px-2.5 py-1 rounded-lg text-xs font-semibold transition-all {statusFilter === 'all' ? 'bg-neutral-800 text-neutral-100 shadow-sm' : 'text-neutral-400'}"
								>All</button>
								<button
									onclick={() => (statusFilter = 'running')}
									class="px-2.5 py-1 rounded-lg text-xs font-semibold transition-all {statusFilter === 'running' ? 'bg-emerald-500/20 text-emerald-300' : 'text-neutral-400'}"
								>Running</button>
								<button
									onclick={() => (statusFilter = 'stopped')}
									class="px-2.5 py-1 rounded-lg text-xs font-semibold transition-all {statusFilter === 'stopped' ? 'bg-rose-500/20 text-rose-300' : 'text-neutral-400'}"
								>Stopped</button>
							</div>
						</div>
					</div>

					<div class="rounded-2xl border border-neutral-800 bg-neutral-900/60 overflow-hidden shadow-xl">
						<div class="overflow-x-auto">
							<table class="w-full text-left border-collapse">
								<thead>
									<tr class="border-b border-neutral-800 bg-neutral-900/80 text-xs font-medium text-neutral-400 uppercase tracking-wider">
										<th class="py-4 px-6">Container Name</th>
										<th class="py-4 px-4">Status</th>
										<th class="py-4 px-6">Image</th>
										<th class="py-4 px-4">Port Mapping</th>
										<th class="py-4 px-4">CPU / RAM</th>
										<th class="py-4 px-6 text-right">Actions</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-neutral-800/60 text-sm">
									{#if loading && containers.length === 0}
										<tr>
											<td colspan="6" class="py-12 text-center text-neutral-500">
												<div class="flex flex-col items-center gap-2">
													<svg class="w-6 h-6 animate-spin text-emerald-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
													<span>Fetching live containers...</span>
												</div>
											</td>
										</tr>
									{:else if filteredContainers.length === 0}
										<tr>
											<td colspan="6" class="py-12 text-center text-neutral-500">
												No containers found.
											</td>
										</tr>
									{:else}
										{#each filteredContainers as container (container.id)}
											<tr class="hover:bg-neutral-800/30 transition-colors group">
												<td class="py-4 px-6 font-medium text-neutral-100">
													<div class="flex items-center gap-3">
														<div class="w-2.5 h-2.5 rounded-full {container.status === 'running' ? 'bg-emerald-400 shadow-sm shadow-emerald-400' : 'bg-neutral-600'}"></div>
														<div>
															<div class="font-semibold text-neutral-100 group-hover:text-emerald-400 transition-colors">{container.name}</div>
															<div class="text-xs text-neutral-500 font-mono">{container.id}</div>
														</div>
													</div>
												</td>

												<td class="py-4 px-4">
													<span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold capitalize border {container.status === 'running'
														? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
														: 'bg-neutral-800 text-neutral-400 border-neutral-700'}">
														{container.status}
													</span>
												</td>

												<td class="py-4 px-6 text-neutral-300 font-mono text-xs">
													{container.image}
												</td>

												<td class="py-4 px-4 text-neutral-300 font-mono text-xs">
													{container.port || 'None'}
												</td>

												<td class="py-4 px-4 font-mono text-xs text-neutral-300">
													{#if container.status === 'running'}
														<div>{container.cpuPercent}% CPU</div>
														<div class="text-neutral-500">{container.memoryMb} MB</div>
													{:else}
														<span class="text-neutral-600">—</span>
													{/if}
												</td>

												<td class="py-4 px-6 text-right">
													<div class="flex items-center justify-end gap-2">
														<button
															onclick={() => openLogStream(container)}
															class="px-2.5 py-1.5 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-300 text-xs font-medium border border-neutral-700/60 transition-all"
														>
															Logs
														</button>

														<button
															onclick={() => toggleContainerStatus(container)}
															disabled={actionLoading === container.id}
															class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-all border disabled:opacity-50 {container.status === 'running'
																? 'bg-rose-500/10 text-rose-400 border-rose-500/20 hover:bg-rose-500/20'
																: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20 hover:bg-emerald-500/20'}"
														>
															{#if actionLoading === container.id}
																Wait...
															{:else}
																{container.status === 'running' ? 'Stop' : 'Start'}
															{/if}
														</button>

														<!-- Delete Container Button -->
														<button
															onclick={() => promptDeleteContainer(container)}
															title="Delete Container"
															class="p-1.5 rounded-lg bg-neutral-800 hover:bg-rose-600/30 text-neutral-400 hover:text-rose-400 border border-neutral-700/60 hover:border-rose-500/30 transition-all"
														>
															<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
														</button>
													</div>
												</td>
											</tr>
										{/each}
									{/if}
								</tbody>
							</table>
						</div>
					</div>
				</section>
			{/if}

			<!-- TAB 3: VOLUMES PAGE -->
			{#if activeTab === 'volumes'}
				<section class="space-y-4">
					<div class="flex items-center justify-between">
						<h3 class="text-lg font-bold text-neutral-100">Docker Volumes</h3>
						<span class="text-xs text-neutral-400 font-mono">{volumes.length} Volumes found</span>
					</div>

					<div class="rounded-2xl border border-neutral-800 bg-neutral-900/60 overflow-hidden shadow-xl">
						<div class="overflow-x-auto">
							<table class="w-full text-left border-collapse">
								<thead>
									<tr class="border-b border-neutral-800 bg-neutral-900/80 text-xs font-medium text-neutral-400 uppercase tracking-wider">
										<th class="py-4 px-6">Volume Name</th>
										<th class="py-4 px-4">Driver</th>
										<th class="py-4 px-6">Mountpoint Path</th>
										<th class="py-4 px-4">Scope</th>
										<th class="py-4 px-6 text-right">Actions</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-neutral-800/60 text-sm">
									{#if loading && volumes.length === 0}
										<tr>
											<td colspan="5" class="py-12 text-center text-neutral-500">
												Fetching volumes from Docker engine...
											</td>
										</tr>
									{:else if volumes.length === 0}
										<tr>
											<td colspan="5" class="py-12 text-center text-neutral-500">
												No Docker volumes found on server.
											</td>
										</tr>
									{:else}
										{#each volumes as volume}
											<tr class="hover:bg-neutral-800/30 transition-colors">
												<td class="py-4 px-6 font-medium text-neutral-100 font-mono text-xs">
													{volume.Name}
												</td>
												<td class="py-4 px-4 text-xs font-mono text-emerald-400">
													{volume.Driver || 'local'}
												</td>
												<td class="py-4 px-6 text-xs font-mono text-neutral-400">
													{volume.Mountpoint}
												</td>
												<td class="py-4 px-4 text-xs font-mono text-neutral-400">
													{volume.Scope || 'local'}
												</td>
												<td class="py-4 px-6 text-right">
													<button
														onclick={() => promptDeleteVolume(volume)}
														title="Delete Volume"
														class="px-3 py-1.5 rounded-lg bg-neutral-800 hover:bg-rose-600/30 text-rose-400 text-xs font-semibold border border-neutral-700/60 hover:border-rose-500/30 transition-all"
													>
														Delete
													</button>
												</td>
											</tr>
										{/each}
									{/if}
								</tbody>
							</table>
						</div>
					</div>
				</section>
			{/if}

			<!-- TAB 4: SETTINGS PAGE -->
			{#if activeTab === 'settings'}
				<section class="space-y-6 max-w-4xl">
					<h3 class="text-lg font-bold text-neutral-100">DevPanel Server Settings</h3>

					<div class="space-y-4">
						<!-- Auto Refresh Rate Controls -->
						<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-3">
							<h4 class="font-semibold text-neutral-200 text-sm">Dashboard Auto-Refresh Interval</h4>
							<p class="text-xs text-neutral-400">Configure how frequently telemetry and Docker stats update.</p>
							<div class="flex items-center gap-2 pt-2">
								{#each [2, 5, 10] as rate}
									<button
										onclick={() => (autoRefreshRateSec = rate)}
										class="px-3.5 py-1.5 rounded-lg text-xs font-mono font-semibold transition-all border {autoRefreshRateSec === rate ? 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30' : 'bg-neutral-800 text-neutral-400 border-neutral-700'}"
									>{rate}s</button>
								{/each}
							</div>
						</div>

						<!-- System Prune Action -->
						<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-3">
							<div class="flex items-center justify-between">
								<div>
									<h4 class="font-semibold text-neutral-200 text-sm">Prune Unused Docker System Resources</h4>
									<p class="text-xs text-neutral-400 mt-1">Remove all stopped containers and dangling volumes to free disk space.</p>
								</div>
								<button
									onclick={handlePruneSystem}
									disabled={actionLoading === 'prune'}
									class="px-4 py-2 rounded-xl bg-rose-600/20 text-rose-400 hover:bg-rose-600 hover:text-white border border-rose-500/30 text-xs font-semibold transition-all disabled:opacity-50"
								>
									{actionLoading === 'prune' ? 'Pruning...' : 'Prune Unused'}
								</button>
							</div>
						</div>

						<!-- Socket Activation Settings -->
						<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-3">
							<div class="flex items-center justify-between">
								<h4 class="font-semibold text-neutral-200 text-sm">Systemd Socket Activation</h4>
								<span class="px-2.5 py-1 rounded-full text-xs font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">Enabled</span>
							</div>
							<p class="text-xs text-neutral-400">Listens on port 80/443 via systemd sockets (`LISTEN_FDS=1`). Server scales to zero after 5 minutes of idle requests.</p>
						</div>

						<!-- Caddy Reverse Proxy Config -->
						<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-3">
							<h4 class="font-semibold text-neutral-200 text-sm">Caddy Admin API Integration</h4>
							<p class="text-xs text-neutral-400">Dynamic reverse-proxy route injection and On-Demand TLS handshake verification.</p>
							<div class="pt-2 text-xs font-mono text-neutral-300">
								<span class="text-neutral-500">Admin Endpoint:</span> http://localhost:2019
							</div>
						</div>

						<!-- SQLite Database -->
						<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-3">
							<h4 class="font-semibold text-neutral-200 text-sm">Pure-Go SQLite Storage</h4>
							<p class="text-xs text-neutral-400">WAL mode enabled for concurrent read/write operations without CGo dependencies.</p>
							<div class="pt-2 text-xs font-mono text-neutral-300">
								<span class="text-neutral-500">Database File:</span> ./devpnl.db
							</div>
						</div>
					</div>
				</section>
			{/if}

			<!-- DELETE CONFIRMATION MODAL -->
			{#if deleteTarget}
				<div class="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
					<div class="bg-neutral-900 border border-neutral-800 rounded-2xl max-w-md w-full p-6 space-y-5 shadow-2xl">
						<div class="flex items-center gap-3">
							<div class="w-10 h-10 rounded-full bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-400">
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
							</div>
							<div>
								<h3 class="font-bold text-neutral-100">Confirm Deletion</h3>
								<p class="text-xs text-neutral-400">Action cannot be undone</p>
							</div>
						</div>

						<p class="text-sm text-neutral-300">
							Are you sure you want to permanently delete {deleteTarget.type} <strong class="text-neutral-100 font-mono">{deleteTarget.label}</strong>?
						</p>

						<div class="flex items-center gap-2 pt-1">
							<input
								type="checkbox"
								id="forceCheck"
								bind:checked={forceDelete}
								class="rounded bg-neutral-800 border-neutral-700 text-emerald-500 focus:ring-emerald-500/30"
							/>
							<label for="forceCheck" class="text-xs text-neutral-400 select-none cursor-pointer">Force deletion (force kill if running / in-use)</label>
						</div>

						<div class="flex items-center justify-end gap-3 pt-3 border-t border-neutral-800">
							<button
								onclick={() => (deleteTarget = null)}
								class="px-4 py-2 rounded-xl bg-neutral-800 hover:bg-neutral-700 text-neutral-300 text-xs font-semibold transition-all"
							>
								Cancel
							</button>
							<button
								onclick={confirmDelete}
								class="px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold transition-all shadow-sm shadow-rose-950"
							>
								Confirm Delete
							</button>
						</div>
					</div>
				</div>
			{/if}

			<!-- ERROR ALERT POPUP MODAL -->
			{#if errorModal}
				<div class="fixed inset-0 bg-black/85 backdrop-blur-md flex items-center justify-center p-4 z-50">
					<div class="bg-neutral-900 border border-rose-900/60 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-2xl animate-in fade-in zoom-in duration-200">
						<div class="flex items-start gap-4">
							<div class="w-10 h-10 rounded-full bg-rose-500/10 border border-rose-500/30 flex items-center justify-center text-rose-400 shrink-0">
								<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
							</div>
							<div class="space-y-1">
								<h3 class="font-bold text-rose-300 text-base">{errorModal.title}</h3>
								<p class="text-xs text-neutral-300 leading-relaxed">{errorModal.message}</p>
							</div>
						</div>

						{#if errorModal.details}
							<div class="bg-neutral-950 p-3 rounded-xl border border-neutral-800 text-xs font-mono text-rose-400/90 overflow-x-auto max-h-36">
								{errorModal.details}
							</div>
						{/if}

						<div class="flex items-center justify-end pt-2">
							<button
								onclick={closeErrorPopup}
								class="px-5 py-2 rounded-xl bg-neutral-800 hover:bg-neutral-700 text-neutral-100 text-xs font-semibold border border-neutral-700/60 transition-all"
							>
								Cancel & Dismiss
							</button>
						</div>
					</div>
				</div>
			{/if}

			<!-- WebSocket Live Log Stream Drawer -->
			{#if selectedContainerLogs}
				<div class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
					<div class="bg-neutral-900 border border-neutral-800 rounded-2xl max-w-3xl w-full p-6 space-y-4 shadow-2xl">
						<div class="flex items-center justify-between border-b border-neutral-800 pb-3">
							<h3 class="font-bold text-neutral-100 flex items-center gap-2">
								<span class="text-emerald-400 font-mono">>_</span> Live WebSocket Logs: {selectedContainerLogs.name}
							</h3>
							<button onclick={closeLogStream} class="text-neutral-400 hover:text-neutral-200 text-sm font-bold">✕</button>
						</div>
						<div class="bg-neutral-950 p-4 rounded-xl font-mono text-xs text-neutral-300 space-y-1.5 max-h-96 overflow-y-auto border border-neutral-800">
							{#each selectedContainerLogs.logs as logLine}
								<div class="hover:bg-neutral-900/50 px-1 py-0.5 rounded">{logLine}</div>
							{/each}
						</div>
					</div>
				</div>
			{/if}
		</div>
	</main>
</div>
