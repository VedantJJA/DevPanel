<script lang="ts">
	import { onMount } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import type { Volume, SystemStats } from '$lib/types';

	let volumes = $state<Volume[]>([]);
	let loading = $state(true);
	let systemStats = $state<SystemStats>({ totalContainers: 0, activeContainers: 0, stoppedContainers: 0, totalMemMb: 0, usedMemMb: 0, memPercent: 0, cpus: 1 });

	const snapshots = [
		{ name: 'auto-backup-2026-08-01-03-00', size: '—', created: 'today 03:00', volume: 'devpnl-data' }
	];

	async function fetchData() {
		try {
			const [vRes, sRes] = await Promise.all([fetch('/api/volumes'), fetch('/api/system/stats')]);
			if (vRes.ok) { const d = await vRes.json(); volumes = d.volumes || []; }
			if (sRes.ok) { const s = await sRes.json(); systemStats = { ...systemStats, ...s }; }
		} catch (e) { console.error(e); } finally { loading = false; }
	}

	onMount(fetchData);
</script>

<AppShell {systemStats}>
	<!-- Header -->
	<div class="mb-6">
		<div class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<h1 class="text-[28px] font-bold leading-tight lg:text-[32px]" style="color: var(--on-surface)">Storage Volumes</h1>
				<p class="mt-1" style="color: var(--on-surface-variant)">Persistent disks attached to containers on this host.</p>
			</div>
			<div class="flex flex-wrap gap-2">
				<button class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
					<span class="material-symbols-outlined" style="font-size: 20px">backup</span>Snapshot all
				</button>
				<button class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
					<span class="material-symbols-outlined" style="font-size: 20px">add</span>New volume
				</button>
			</div>
		</div>
	</div>

	<!-- Stat Cards -->
	<div class="mb-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Volumes</span>
			<span class="text-2xl font-bold" style="color: var(--primary)">{volumes.length}</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Driver</span>
			<span class="text-2xl font-bold">local</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Snapshots</span>
			<span class="text-2xl font-bold">—</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">manual only</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Scope</span>
			<span class="text-2xl font-bold">local</span>
		</div>
	</div>

	<!-- Volumes Grid -->
	{#if loading}
		<div class="flex items-center justify-center py-16">
			<span class="material-symbols-outlined animate-spin" style="color: var(--primary); font-size: 32px">refresh</span>
		</div>
	{:else if volumes.length === 0}
		<div class="card-surface flex flex-col items-center py-16 text-center">
			<span class="material-symbols-outlined mb-3" style="font-size: 48px; color: var(--outline)">storage</span>
			<h2 class="font-semibold">No volumes found</h2>
			<p class="mt-2 text-sm" style="color: var(--on-surface-variant)">Create a volume or deploy a project to get started.</p>
		</div>
	{:else}
		<div class="mb-6 grid grid-cols-1 gap-6 md:grid-cols-2">
			{#each volumes as v}
				<div class="card-surface p-5">
					<div class="mb-3 flex items-start justify-between">
						<div class="flex items-center gap-3">
							<span class="flex h-10 w-10 items-center justify-center rounded-lg" style="background-color: var(--surface-low); color: var(--primary)">
								<span class="material-symbols-outlined">hard_drive</span>
							</span>
							<div>
								<h2 class="font-bold">{v.Name}</h2>
								<p class="font-mono text-xs" style="color: var(--on-surface-variant)">{v.Mountpoint || '—'}</p>
							</div>
						</div>
						<span class="label-caps rounded px-2 py-1" style="background-color: var(--surface-low); color: var(--on-surface-variant)">{v.Driver || 'local'}</span>
					</div>
					<div class="mt-4 flex items-center justify-between border-t pt-3 text-xs" style="border-color: var(--outline-variant); color: var(--on-surface-variant)">
						<span>Scope: {v.Scope || 'local'}</span>
						{#if v.CreatedAt}
							<span>Created {v.CreatedAt.slice(0, 10)}</span>
						{/if}
						<div class="flex gap-1">
							<button class="rounded p-1 transition-colors hover:opacity-80" style="color: var(--error)"
								onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--error-container)'; }}
								onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}>
								<span class="material-symbols-outlined" style="font-size: 18px">delete</span>
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Snapshots -->
	<section class="card-surface overflow-hidden">
		<div class="border-b px-5 py-4" style="border-color: var(--outline-variant)">
			<h2 class="font-bold">Recent Snapshots</h2>
		</div>
		<ul class="divide-y" style="border-color: var(--outline-variant)">
			{#each snapshots as s}
				<li class="flex items-center justify-between gap-4 px-5 py-3">
					<div>
						<p class="font-mono text-sm">{s.name}</p>
						<p class="text-xs" style="color: var(--on-surface-variant)">{s.volume} · {s.created}</p>
					</div>
					<div class="flex items-center gap-4">
						<span class="text-sm" style="color: var(--on-surface-variant)">{s.size}</span>
						<button class="flex items-center gap-2 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors hover:opacity-80" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
							<span class="material-symbols-outlined" style="font-size: 16px">restore</span>Restore
						</button>
					</div>
				</li>
			{/each}
		</ul>
	</section>
</AppShell>
