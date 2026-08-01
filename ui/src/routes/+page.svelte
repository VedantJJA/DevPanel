<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import type { Container, SystemStats, BlueprintItem } from '$lib/types';

	let containers = $state<Container[]>([]);
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
	let loading = $state(true);
	let pingMs = $state<number | null>(null);
	let autoRefreshSec = $state(5);
	let pollInterval: ReturnType<typeof setInterval> | null = null;
	let pingInterval: ReturnType<typeof setInterval> | null = null;

	let events = $state<{ time: string; kind: 'success' | 'error' | 'warning' | 'info'; text: string }[]>([]);

	const eventColors = {
		success: 'var(--success)',
		error: 'var(--error)',
		warning: 'var(--warning)',
		info: 'var(--primary)'
	};

	const statusStyles: Record<string, { label: string; bg: string; color: string }> = {
		running: { label: 'Running', bg: 'var(--success-container)', color: 'var(--on-success-container)' },
		stopped: { label: 'Stopped', bg: 'var(--surface-high)', color: 'var(--on-surface-variant)' },
		restarting: { label: 'Restarting', bg: 'var(--warning-container)', color: 'var(--warning)' },
		error: { label: 'Error', bg: 'var(--error-container)', color: 'var(--error)' }
	};

	async function measurePing() {
		const start = performance.now();
		try {
			const res = await fetch('/healthz', { cache: 'no-store' });
			if (res.ok) pingMs = Math.round(performance.now() - start);
		} catch {
			pingMs = null;
		}
	}

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

	async function fetchData() {
		try {
			const [containersRes, statsRes, blueprintsRes] = await Promise.all([
				fetch('/api/containers'),
				fetch('/api/system/stats'),
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
					activeContainers: stats.activeContainers ?? containers.filter((c) => c.status === 'running').length,
					stoppedContainers: stats.stoppedContainers ?? containers.filter((c) => c.status !== 'running').length,
					totalMemMb: stats.totalMemMb || 0,
					usedMemMb: stats.usedMemMb || 0,
					memPercent: stats.memPercent || 0,
					cpuPercent: stats.cpuPercent ?? 0,
					cpus: stats.cpus || 1,
					os: stats.os,
					arch: stats.arch
				};
			}
			if (blueprintsRes.ok) {
				const bpData = await blueprintsRes.json();
				blueprints = bpData.blueprints || [];

				const newEvents: { time: string; kind: 'success' | 'error' | 'warning' | 'info'; text: string }[] = [];
				for (const bp of blueprints) {
					const timeStr = formatTimeAgo(bp.created_at || bp.updated_at || '');
					if (bp.status === 'active' || bp.status === 'live' || bp.status === 'ready') {
						newEvents.push({ time: timeStr, kind: 'success', text: `Deploy complete — ${bp.name}` });
					} else if (bp.status === 'error') {
						newEvents.push({ time: timeStr, kind: 'error', text: `Build error — ${bp.name}` });
					} else if (bp.status === 'building' || bp.status === 'deploying') {
						newEvents.push({ time: timeStr, kind: 'info', text: `Redeploy in progress — ${bp.name}` });
					}
				}
				if (newEvents.length === 0) {
					newEvents.push({ time: 'Just now', kind: 'info', text: 'System initialized. Ready for project deployments.' });
				}
				events = newEvents;
			}
		} catch (err) {
			console.error('fetchData error:', err);
		} finally {
			loading = false;
		}
	}

	onMount(async () => {
		await fetchData();
		await measurePing();
		pollInterval = setInterval(fetchData, autoRefreshSec * 1000);
		pingInterval = setInterval(measurePing, 3000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
		if (pingInterval) clearInterval(pingInterval);
	});

	const running = $derived(containers.filter((c) => c.status === 'running').length);
	const cpuPct = $derived(systemStats.cpuPercent ?? 0);
	const memPct = $derived(systemStats.memPercent ?? 0);
</script>

<AppShell {systemStats} {pingMs}>
	<!-- Page Header -->
	<div class="mb-6">
		<div class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<h1 class="text-[28px] font-bold leading-tight lg:text-[32px]" style="color: var(--on-surface)">Dashboard</h1>
				<p class="mt-1" style="color: var(--on-surface-variant)">DevPanel · everything on this machine, in one place.</p>
			</div>
			<div class="flex flex-wrap gap-2">
				<button
					onclick={() => fetchData()}
					class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors hover:opacity-80"
					style="border-color: var(--outline-variant); background-color: var(--surface-lowest);"
				>
					<span class="material-symbols-outlined" style="font-size: 20px">refresh</span>
					Refresh
				</button>
				<a
					href="/new"
					class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-opacity hover:opacity-90"
					style="background-color: var(--primary); color: var(--on-primary);"
				>
					<span class="material-symbols-outlined" style="font-size: 20px">add</span>
					New Service
				</a>
			</div>
		</div>
	</div>

	<!-- Stat Cards -->
	<div class="mb-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
		<!-- Projects -->
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Projects</span>
			<span class="text-2xl font-bold" style="color: var(--primary)">{blueprints.length}</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">{blueprints.filter(b => b.status === 'active').length} active</span>
		</div>
		<!-- Services running -->
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Containers running</span>
			<span class="text-2xl font-bold" style="color: var(--success)">{running}/{containers.length}</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">{containers.filter(c => c.status === 'restarting').length} restarting</span>
		</div>
		<!-- CPU Load -->
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">VM CPU</span>
			<span class="text-2xl font-bold" style="color: var(--on-surface)">{cpuPct.toFixed(1)}%</span>
			<div class="mt-2 h-1.5 w-full rounded-full" style="background-color: var(--surface-high)">
				<div class="h-1.5 rounded-full transition-all" style="width: {cpuPct}%; background-color: var(--primary)"></div>
			</div>
		</div>
		<!-- Memory -->
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">VM Memory</span>
			<span class="text-2xl font-bold" style="color: {memPct > 80 ? 'var(--error)' : 'var(--on-surface)'}">{memPct.toFixed(1)}%</span>
			<div class="mt-2 h-1.5 w-full rounded-full" style="background-color: var(--surface-high)">
				<div class="h-1.5 rounded-full transition-all" style="width: {memPct}%; background-color: {memPct > 80 ? 'var(--error)' : 'var(--primary)'}"></div>
			</div>
		</div>
	</div>

	<!-- Main Grid -->
	<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
		<!-- Active Containers (2/3 width) -->
		<section class="card-surface overflow-hidden lg:col-span-2">
			<div class="flex items-center justify-between border-b px-5 py-4" style="border-color: var(--outline-variant)">
				<h2 class="font-bold">Active containers</h2>
				<a href="/containers" class="text-sm font-medium hover:underline" style="color: var(--primary)">All containers</a>
			</div>
			{#if loading}
				<div class="flex items-center justify-center py-12">
					<span class="material-symbols-outlined animate-spin" style="color: var(--primary)">refresh</span>
				</div>
			{:else if containers.length === 0}
				<div class="py-12 text-center text-sm" style="color: var(--on-surface-variant)">
					No containers found. <a href="/new" class="font-medium hover:underline" style="color: var(--primary)">Deploy your first service →</a>
				</div>
			{:else}
				<ul class="divide-y" style="border-color: var(--outline-variant)">
					{#each containers.slice(0, 6) as container}
						{@const s = statusStyles[container.status] ?? statusStyles.stopped}
						<li>
							<a
								href="/containers"
								class="flex items-center gap-4 px-5 py-4 transition-colors hover:opacity-90"
								onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--surface-low)'; }}
								onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}
							>
								<span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg" style="background-color: var(--surface-low); color: var(--primary)">
									<span class="material-symbols-outlined">deployed_code</span>
								</span>
								<div class="min-w-0 flex-1">
									<p class="truncate font-semibold">{container.name}</p>
									<p class="truncate text-xs" style="color: var(--on-surface-variant)">{container.image}</p>
								</div>
								<div class="hidden text-right text-xs sm:block" style="color: var(--on-surface-variant)">{container.uptime || '—'}</div>
								<span class="inline-block rounded-full px-2 py-1 text-[10px] font-bold uppercase tracking-wider" style="background-color: {s.bg}; color: {s.color}">{s.label}</span>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<!-- Right Column: Resources + Events -->
		<div class="flex flex-col gap-6">
			<!-- VM Resources -->
			<section class="card-surface p-5">
				<h2 class="mb-4 font-bold">VM Resources</h2>
				{#each [
					{ label: 'CPU', value: cpuPct, detail: `${cpuPct.toFixed(1)}% of ${systemStats.cpus} vCPU` },
					{ label: 'Memory', value: memPct, detail: `${Math.round(systemStats.usedMemMb / 1024 * 10) / 10} of ${Math.round(systemStats.totalMemMb / 1024 * 10) / 10} GB` }
				] as r}
					<div class="mb-4 last:mb-0">
						<div class="mb-1 flex justify-between text-xs">
							<span class="font-medium">{r.label}</span>
							<span style="color: var(--on-surface-variant)">{r.detail}</span>
						</div>
						<div class="h-1.5 w-full rounded-full" style="background-color: var(--surface-high)">
							<div class="h-1.5 rounded-full" style="width: {r.value}%; background-color: var(--primary)"></div>
						</div>
					</div>
				{/each}
			</section>

			<!-- Recent Events -->
			<section class="card-surface p-5">
				<h2 class="mb-3 font-bold">Recent Events</h2>
				<ul class="space-y-3">
					{#each events as e}
						<li class="flex gap-3 text-sm">
							<span class="label-caps shrink-0" style="color: var(--on-surface-variant)">{e.time}</span>
							<span style="color: {eventColors[e.kind]}">{e.text}</span>
						</li>
					{/each}
				</ul>
			</section>
		</div>
	</div>

	<!-- Projects Snapshot -->
	<section class="card-surface mt-6 overflow-hidden">
		<div class="flex items-center justify-between border-b px-5 py-4" style="border-color: var(--outline-variant)">
			<h2 class="font-bold">Hosted projects</h2>
			<a href="/blueprints" class="text-sm font-medium hover:underline" style="color: var(--primary)">All projects</a>
		</div>
		{#if blueprints.length === 0}
			<div class="py-10 text-center text-sm" style="color: var(--on-surface-variant)">
				No projects yet. <a href="/new" class="font-medium hover:underline" style="color: var(--primary)">Create your first project →</a>
			</div>
		{:else}
			<div class="grid grid-cols-1 divide-y sm:grid-cols-3 sm:divide-x sm:divide-y-0" style="border-color: var(--outline-variant)">
				{#each blueprints.slice(0, 3) as bp}
					{@const s = statusStyles[bp.status ?? 'stopped'] ?? statusStyles.stopped}
					<a href={`/projects/${bp.id}`} class="block p-5 transition-colors hover:bg-[color:var(--surface-low)]">
						<div class="mb-2 flex items-center justify-between">
							<span class="label-caps rounded px-2 py-1" style="background-color: var(--surface-low); color: var(--primary)">{bp.id?.slice(0, 8) || bp.name}</span>
							<span class="inline-block rounded-full px-2 py-1 text-[10px] font-bold uppercase tracking-wider" style="background-color: {s.bg}; color: {s.color}">{bp.status || 'Unknown'}</span>
						</div>
						<p class="font-semibold">{bp.name}</p>
						<p class="text-xs" style="color: var(--on-surface-variant)">{bp.serviceCount ?? bp.service_count_actual ?? 0} services</p>
					</a>
				{/each}
			</div>
		{/if}
	</section>
</AppShell>
