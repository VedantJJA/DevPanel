<script lang="ts">
	import { onMount } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import type { Container, SystemStats } from '$lib/types';

	let containers = $state<Container[]>([]);
	let systemStats = $state<SystemStats>({ totalContainers: 0, activeContainers: 0, stoppedContainers: 0, totalMemMb: 0, usedMemMb: 0, memPercent: 0, cpus: 1 });
	let loading = $state(true);
	let actionLoading = $state<string | null>(null);
	let pingMs = $state<number | null>(null);

	const statusStyles: Record<string, { label: string; bg: string; color: string }> = {
		running: { label: 'Running', bg: 'var(--success-container)', color: 'var(--on-success-container)' },
		stopped: { label: 'Stopped', bg: 'var(--surface-high)', color: 'var(--on-surface-variant)' },
		restarting: { label: 'Restarting', bg: 'var(--warning-container)', color: 'var(--warning)' },
		error: { label: 'Error', bg: 'var(--error-container)', color: 'var(--error)' }
	};

	async function fetchData() {
		try {
			const [cRes, sRes] = await Promise.all([fetch('/api/containers'), fetch('/api/system/stats')]);
			if (cRes.ok) { const d = await cRes.json(); containers = d.containers || []; }
			if (sRes.ok) { const s = await sRes.json(); systemStats = { ...systemStats, ...s }; }
		} catch (e) { console.error(e); } finally { loading = false; }
	}

	async function measurePing() {
		const start = performance.now();
		try { const r = await fetch('/healthz', { cache: 'no-store' }); if (r.ok) pingMs = Math.round(performance.now() - start); } catch { pingMs = null; }
	}

	async function toggleContainer(c: Container) {
		actionLoading = c.id;
		const action = c.status === 'running' ? 'stop' : 'start';
		try {
			await fetch(`/api/containers/${action}?id=${c.id}`, { method: 'POST' });
			await fetchData();
		} catch (e) { console.error(e); } finally { actionLoading = null; }
	}

	async function deleteContainer(c: Container) {
		if (!confirm(`Delete container "${c.name}"? This cannot be undone.`)) return;
		actionLoading = c.id;
		try {
			await fetch(`/api/containers/delete?id=${c.id}&force=true`, { method: 'DELETE' });
			await fetchData();
		} catch (e) { console.error(e); } finally { actionLoading = null; }
	}

	let logModal = $state<{ id: string; name: string; logs: string[] } | null>(null);
	let logSocket: WebSocket | null = null;

	function openLogs(c: Container) {
		if (logSocket) { logSocket.close(); logSocket = null; }
		logModal = { id: c.id, name: c.name, logs: [`[SYS] Connecting to log stream for ${c.name}…`] };
		const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		logSocket = new WebSocket(`${proto}//${window.location.host}/ws/logs?id=${c.id}&tail=100`);
		logSocket.onmessage = (e) => {
			try {
				const msg = JSON.parse(e.data);
				if (msg.type === 'log' && Array.isArray(msg.data)) {
					const lines = msg.data.map((l: any) => `${l.timestamp ? `[${l.timestamp}] ` : ''}${l.line}`);
					if (logModal) logModal = { ...logModal, logs: [...logModal.logs, ...lines] };
				}
			} catch { if (logModal) logModal = { ...logModal, logs: [...logModal.logs, e.data] }; }
		};
		logSocket.onclose = () => { if (logModal) logModal = { ...logModal, logs: [...logModal.logs, '[SYS] Log stream closed.'] }; };
	}

	onMount(async () => { await fetchData(); await measurePing(); });

	const running = $derived(containers.filter(c => c.status === 'running').length);
</script>

<AppShell {systemStats} {pingMs}>
	<!-- Header -->
	<div class="mb-6">
		<div class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<h1 class="text-[28px] font-bold leading-tight lg:text-[32px]" style="color: var(--on-surface)">Raw Containers</h1>
				<p class="mt-1" style="color: var(--on-surface-variant)">Managing {containers.length} containers on this host.</p>
			</div>
			<div class="flex flex-wrap gap-2">
				<button onclick={fetchData} class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
					<span class="material-symbols-outlined" style="font-size: 20px">filter_list</span>Filters
				</button>
				<a href="/new" class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
					<span class="material-symbols-outlined" style="font-size: 20px">add</span>Run container
				</a>
			</div>
		</div>
	</div>

	<!-- Stat Cards -->
	<div class="mb-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Total containers</span>
			<span class="text-2xl font-bold" style="color: var(--primary)">{containers.length}</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Running</span>
			<span class="text-2xl font-bold" style="color: var(--success)">{running}</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">{containers.length > 0 ? Math.round((running / containers.length) * 100) : 0}% availability</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">CPU</span>
			<span class="text-2xl font-bold">{(systemStats.cpuPercent ?? 0).toFixed(1)}%</span>
			<div class="mt-2 h-1.5 w-full rounded-full" style="background-color: var(--surface-high)">
				<div class="h-1.5 rounded-full" style="width: {systemStats.cpuPercent ?? 0}%; background-color: var(--primary)"></div>
			</div>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Stopped/Error</span>
			<span class="text-2xl font-bold" style="color: var(--error)">{containers.filter(c => c.status !== 'running').length}</span>
		</div>
	</div>

	<!-- Containers Table -->
	<section class="card-surface overflow-hidden">
		{#if loading}
			<div class="flex items-center justify-center py-16">
				<span class="material-symbols-outlined animate-spin" style="color: var(--primary); font-size: 32px">refresh</span>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-left">
					<thead>
						<tr class="border-b" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<th class="label-caps px-5 py-3" style="color: var(--on-surface-variant)">Container</th>
							<th class="label-caps px-5 py-3" style="color: var(--on-surface-variant)">Image</th>
							<th class="label-caps px-5 py-3 text-center" style="color: var(--on-surface-variant)">Status</th>
							<th class="label-caps px-5 py-3" style="color: var(--on-surface-variant)">Port</th>
							<th class="label-caps px-5 py-3" style="color: var(--on-surface-variant)">Usage</th>
							<th class="label-caps px-5 py-3 text-right" style="color: var(--on-surface-variant)">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y" style="border-color: var(--outline-variant)">
						{#each containers as c}
							{@const s = statusStyles[c.status] ?? statusStyles.stopped}
							<tr class="transition-colors" style={c.status === 'stopped' ? 'opacity: 0.7' : ''}
								onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--surface-low)'; }}
								onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}>
								<td class="px-5 py-4">
									<span class="label-caps rounded px-2 py-1" style="background-color: var(--surface-low); color: var(--primary)">{c.id?.slice(0, 12)}</span>
								</td>
								<td class="px-5 py-4">
									<p class="text-sm font-bold">{c.name}</p>
									<p class="text-xs" style="color: var(--on-surface-variant)">{c.image}</p>
								</td>
								<td class="px-5 py-4 text-center">
									<span class="inline-block rounded-full px-2 py-1 text-[10px] font-bold uppercase tracking-wider" style="background-color: {s.bg}; color: {s.color}">{s.label}</span>
									<p class="mt-1 text-[10px]" style="color: var(--on-surface-variant)">{c.uptime || '—'}</p>
								</td>
								<td class="px-5 py-4 font-mono text-xs">{c.port || '—'}</td>
								<td class="px-5 py-4">
									<div class="w-32">
										<div class="mb-1 flex justify-between text-[10px]" style="color: var(--on-surface-variant)">
											<span>CPU {(c.cpuPercent ?? 0).toFixed(1)}%</span>
											<span>MEM {Math.round(c.memoryMb)}MB</span>
										</div>
										<div class="h-1 w-full overflow-hidden rounded-full" style="background-color: var(--surface-high)">
											<div class="h-full" style="width: {Math.min(c.cpuPercent ?? 0, 100)}%; background-color: var(--primary)"></div>
										</div>
									</div>
								</td>
								<td class="px-5 py-4">
									<div class="flex justify-end gap-1">
										<button onclick={() => openLogs(c)} title="Logs" class="rounded p-1.5 transition-colors" style="color: var(--on-surface-variant)"
											onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--surface-high)'; }}
											onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}>
											<span class="material-symbols-outlined" style="font-size: 20px">terminal</span>
										</button>
										<button onclick={() => toggleContainer(c)} disabled={actionLoading === c.id} title={c.status === 'running' ? 'Stop' : 'Start'} class="rounded p-1.5 transition-colors" style="color: var(--on-surface-variant)"
											onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--surface-high)'; }}
											onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}>
											<span class="material-symbols-outlined" style="font-size: 20px">{c.status === 'running' ? 'pause' : 'play_arrow'}</span>
										</button>
										<button onclick={() => deleteContainer(c)} disabled={actionLoading === c.id} title="Delete" class="rounded p-1.5 transition-colors" style="color: var(--error)"
											onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--error-container)'; }}
											onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}>
											<span class="material-symbols-outlined" style="font-size: 20px">delete</span>
										</button>
									</div>
								</td>
							</tr>
						{/each}
						{#if containers.length === 0}
							<tr>
								<td colspan="6" class="px-5 py-12 text-center text-sm" style="color: var(--on-surface-variant)">
									No containers found. <a href="/new" style="color: var(--primary)" class="font-medium hover:underline">Deploy your first project →</a>
								</td>
							</tr>
						{/if}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
</AppShell>

<!-- Log Stream Modal -->
{#if logModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4" style="background-color: rgba(0,0,0,0.5)">
		<div class="card-surface flex w-full max-w-3xl flex-col overflow-hidden" style="max-height: 80vh">
			<div class="flex items-center justify-between border-b px-5 py-3" style="border-color: var(--outline-variant)">
				<h2 class="font-bold">Logs — {logModal.name}</h2>
				<button onclick={() => { logSocket?.close(); logModal = null; }} class="rounded p-1 transition-colors hover:opacity-80" style="color: var(--on-surface-variant)">
					<span class="material-symbols-outlined">close</span>
				</button>
			</div>
			<pre class="flex-1 overflow-auto p-5 font-mono text-xs leading-6" style="background-color: var(--inverse-surface); color: var(--inverse-on-surface); max-height: 60vh">{logModal.logs.join('\n')}</pre>
		</div>
	</div>
{/if}
