<script lang="ts">
	import { onMount } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import type { BlueprintItem, SystemStats } from '$lib/types';
	import { routingConfig, loadRoutingConfig } from '$lib/stores/routing';
	import { buildProjectUrl } from '$lib/shared/url';


	let blueprints = $state<BlueprintItem[]>([]);
	let loading = $state(true);
	let actionLoading = $state<string | null>(null);
	let systemStats = $state<SystemStats>({ totalContainers: 0, activeContainers: 0, stoppedContainers: 0, totalMemMb: 0, usedMemMb: 0, memPercent: 0, cpus: 1 });

	const statusStyles: Record<string, { label: string; bg: string; color: string }> = {
		active: { label: 'Active', bg: 'var(--success-container)', color: 'var(--on-success-container)' },
		live: { label: 'Live', bg: 'var(--success-container)', color: 'var(--on-success-container)' },
		building: { label: 'Building', bg: 'var(--primary-fixed)', color: 'var(--primary)' },
		deploying: { label: 'Deploying', bg: 'var(--primary-fixed)', color: 'var(--primary)' },
		valid: { label: 'Valid', bg: 'var(--primary-fixed)', color: 'var(--primary)' },
		ready: { label: 'Ready', bg: 'var(--surface-high)', color: 'var(--on-surface-variant)' },
		error: { label: 'Build Error', bg: '#fee2e2', color: '#dc2626' },
		failed: { label: 'Deploy Failed', bg: '#fee2e2', color: '#dc2626' }
	};

	async function fetchData() {
		try {
			const [bpRes, sRes] = await Promise.all([fetch('/api/blueprints'), fetch('/api/system/stats')]);
			if (bpRes.ok) { const d = await bpRes.json(); blueprints = d.blueprints || []; }
			if (sRes.ok) { const s = await sRes.json(); systemStats = { ...systemStats, ...s }; }
		} catch (e) { console.error(e); } finally { loading = false; }
	}

	async function deployBlueprint(bp: BlueprintItem) {
		actionLoading = bp.id;
		try {
			const res = await fetch('/api/blueprints/deploy', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ repo_url: bp.repo_url, app_name: bp.name })
			});
			const data = await res.json();
			if (!res.ok || data.error) alert(data.error || 'Deploy failed');
			else await fetchData();
		} catch (e: any) { alert(e.message); } finally { actionLoading = null; }
	}

	async function deleteBlueprint(bp: BlueprintItem) {
		if (!confirm(`Delete project "${bp.name}"? This is irreversible.`)) return;
		actionLoading = bp.id;
		try {
			await fetch(`/api/blueprints/delete?id=${encodeURIComponent(bp.id)}`, { method: 'DELETE' });
			await fetchData();
		} catch (e) { console.error(e); } finally { actionLoading = null; }
	}

	onMount(async () => {
		await loadRoutingConfig();
		fetchData();
	});

</script>

<AppShell {systemStats}>
	<!-- Header -->
	<div class="mb-6">
		<div class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<h1 class="text-[28px] font-bold leading-tight lg:text-[32px]" style="color: var(--on-surface)">Projects</h1>
				<p class="mt-1" style="color: var(--on-surface-variant)">{blueprints.length} projects deployed on this host.</p>
			</div>
			<div class="flex flex-wrap gap-2">
				<button onclick={fetchData} class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
					<span class="material-symbols-outlined" style="font-size: 20px">filter_list</span>Filters
				</button>
				<a href="/new" class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
					<span class="material-symbols-outlined" style="font-size: 20px">add</span>New Project
				</a>
			</div>
		</div>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<span class="material-symbols-outlined animate-spin" style="color: var(--primary); font-size: 36px">refresh</span>
		</div>
	{:else if blueprints.length === 0}
		<div class="card-surface flex flex-col items-center py-20 text-center">
			<span class="material-symbols-outlined mb-4" style="font-size: 48px; color: var(--outline)">folder_open</span>
			<h2 class="text-lg font-semibold">No projects yet</h2>
			<p class="mt-2 text-sm" style="color: var(--on-surface-variant)">Create your first project to get started.</p>
			<a href="/new" class="mt-6 flex items-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
				<span class="material-symbols-outlined" style="font-size: 20px">add</span>Create project
			</a>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
			{#each blueprints as bp}
				{@const s = statusStyles[bp.status ?? 'ready'] ?? statusStyles.ready}
				<div class="card-surface flex flex-col p-5 transition-all hover:border-[color:var(--primary)] hover:shadow-sm">
					<a href={`/projects/${bp.id}`} class="mb-3 flex items-start justify-between gap-3 group">
						<div class="flex items-center gap-3">
							<span class="flex h-10 w-10 items-center justify-center rounded-lg transition-transform group-hover:scale-105" style="background-color: var(--primary); color: var(--on-primary)">
								<span class="material-symbols-outlined">layers</span>
							</span>
							<div>
								<h2 class="font-bold group-hover:underline" style="color: var(--on-surface)">{bp.name}</h2>
								<p class="label-caps" style="color: var(--on-surface-variant)">
									{bp.id?.slice(0, 8) || '—'}
								</p>
							</div>
						</div>
						<span class="inline-block rounded-full px-2 py-1 text-[10px] font-bold uppercase tracking-wider" style="background-color: {s.bg}; color: {s.color}">{s.label}</span>
					</a>

					{#if bp.repo_url}
						<a href={`/projects/${bp.id}`} class="mb-4 truncate text-sm hover:underline" style="color: var(--on-surface-variant)">{bp.repo_url}</a>
					{/if}

					<div class="mt-auto flex items-center justify-between border-t pt-3 text-xs" style="border-color: var(--outline-variant); color: var(--on-surface-variant)">
						<span>{bp.serviceCount ?? bp.service_count_actual ?? 0} services</span>
						<div class="flex gap-2">
							<a
								href={buildProjectUrl({ routingMode: $routingConfig.mode, rootDomain: $routingConfig.baseDomain, projectName: bp.name })}
								target="_blank"
								rel="noopener noreferrer"
								class="flex items-center gap-1 rounded-lg border px-3 py-1.5 text-xs font-semibold transition-colors hover:bg-slate-100"
								style="border-color: var(--outline-variant); color: var(--on-surface)"
							>
								<span class="material-symbols-outlined" style="font-size: 16px">open_in_new</span>
								Open
							</a>
							<button
								onclick={(e) => { e.stopPropagation(); deployBlueprint(bp); }}
								disabled={actionLoading === bp.id}
								class="flex items-center gap-1 rounded-lg px-3 py-1.5 text-xs font-semibold transition-opacity hover:opacity-80 disabled:opacity-50"
								style="background-color: var(--primary); color: var(--on-primary)"
							>
								<span class="material-symbols-outlined" style="font-size: 16px">rocket_launch</span>
								{actionLoading === bp.id ? 'Deploying…' : 'Deploy'}
							</button>
							<button
								onclick={(e) => { e.stopPropagation(); deleteBlueprint(bp); }}
								disabled={actionLoading === bp.id}
								class="rounded-lg p-1.5 transition-colors hover:opacity-80 disabled:opacity-50"
								style="color: var(--error)"
								onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--error-container)'; }}
								onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}
							>
								<span class="material-symbols-outlined" style="font-size: 18px">delete</span>
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</AppShell>
