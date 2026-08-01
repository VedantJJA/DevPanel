<script lang="ts">
	import { onMount } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import type { SystemStats } from '$lib/types';

	let systemStats = $state<SystemStats>({ totalContainers: 0, activeContainers: 0, stoppedContainers: 0, totalMemMb: 0, usedMemMb: 0, memPercent: 0, cpus: 1 });

	// Caddy proxy routes — these would come from the API in a real impl
	// For now wired to the proxy config API endpoint when available
	let routes = $state<{ host: string; upstream: string; tls: string; status: string; requests: string }[]>([]);
	let loading = $state(true);

	const caddyfile = `# DevPanel managed Caddyfile
# Auto-generated from routing config

:80 {
    reverse_proxy localhost:8090
}`;

	async function fetchData() {
		try {
			const sRes = await fetch('/api/system/stats');
			if (sRes.ok) { const s = await sRes.json(); systemStats = { ...systemStats, ...s }; }
			// Try to fetch caddy routes if endpoint exists
			try {
				const cRes = await fetch('/api/proxy/routes');
				if (cRes.ok) { const d = await cRes.json(); routes = d.routes || []; }
			} catch { routes = []; }
		} catch (e) { console.error(e); } finally { loading = false; }
	}

	onMount(fetchData);
</script>

<AppShell {systemStats}>
	<!-- Header -->
	<div class="mb-6">
		<div class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<h1 class="text-[28px] font-bold leading-tight lg:text-[32px]" style="color: var(--on-surface)">Caddy Reverse Proxy</h1>
				<p class="mt-1" style="color: var(--on-surface-variant)">Every public hostname on this host, its upstream and TLS state.</p>
			</div>
			<div class="flex flex-wrap gap-2">
				<button onclick={fetchData} class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
					<span class="material-symbols-outlined" style="font-size: 20px">restart_alt</span>Reload config
				</button>
				<button class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold" style="background-color: var(--primary); color: var(--on-primary)">
					<span class="material-symbols-outlined" style="font-size: 20px">add</span>Add route
				</button>
			</div>
		</div>
	</div>

	<!-- Stat Cards -->
	<div class="mb-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Routes</span>
			<span class="text-2xl font-bold" style="color: var(--primary)">{routes.length || '—'}</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Certificates</span>
			<span class="text-2xl font-bold" style="color: var(--success)">Auto-TLS</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Requests (24h)</span>
			<span class="text-2xl font-bold">—</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Proxy</span>
			<span class="text-2xl font-bold">Caddy</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">Managed</span>
		</div>
	</div>

	<!-- Routes Table -->
	<section class="card-surface mb-6 overflow-hidden">
		{#if loading}
			<div class="flex items-center justify-center py-12">
				<span class="material-symbols-outlined animate-spin" style="color: var(--primary); font-size: 28px">refresh</span>
			</div>
		{:else if routes.length === 0}
			<div class="flex flex-col items-center py-12 text-center">
				<span class="material-symbols-outlined mb-3" style="font-size: 40px; color: var(--outline)">router</span>
				<p class="text-sm" style="color: var(--on-surface-variant)">No proxy routes configured yet.</p>
				<p class="mt-1 text-xs" style="color: var(--on-surface-variant)">Routes are created automatically when you deploy a web service.</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-left">
					<thead>
						<tr class="border-b" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<th class="label-caps px-5 py-3" style="color: var(--on-surface-variant)">Hostname</th>
							<th class="label-caps px-5 py-3" style="color: var(--on-surface-variant)">Upstream</th>
							<th class="label-caps px-5 py-3" style="color: var(--on-surface-variant)">TLS</th>
							<th class="label-caps px-5 py-3" style="color: var(--on-surface-variant)">Traffic</th>
							<th class="label-caps px-5 py-3 text-right" style="color: var(--on-surface-variant)">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y" style="border-color: var(--outline-variant)">
						{#each routes as r}
							<tr onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--surface-low)'; }}
								onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}>
								<td class="px-5 py-4">
									<span class="flex items-center gap-2 text-sm font-semibold">
										<span class="material-symbols-outlined" style="font-size: 18px; color: {r.status === 'active' ? 'var(--success)' : 'var(--warning)'}">
											{r.status === 'active' ? 'lock' : 'lock_open'}
										</span>
										{r.host}
									</span>
								</td>
								<td class="px-5 py-4 font-mono text-xs">{r.upstream}</td>
								<td class="px-5 py-4 text-sm" style="color: var(--on-surface-variant)">{r.tls}</td>
								<td class="px-5 py-4 text-sm" style="color: var(--on-surface-variant)">{r.requests}</td>
								<td class="px-5 py-4">
									<div class="flex justify-end gap-1">
										<button class="rounded p-1.5 transition-colors" style="color: var(--on-surface-variant)"
											onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--surface-high)'; }}
											onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}>
											<span class="material-symbols-outlined" style="font-size: 20px">edit</span>
										</button>
										<button class="rounded p-1.5 transition-colors" style="color: var(--error)"
											onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--error-container)'; }}
											onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}>
											<span class="material-symbols-outlined" style="font-size: 20px">delete</span>
										</button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	<!-- Caddyfile Preview -->
	<section class="card-surface overflow-hidden">
		<div class="flex items-center justify-between border-b px-5 py-3" style="border-color: var(--outline-variant)">
			<h2 class="font-bold">Generated Caddyfile</h2>
			<button class="flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors hover:opacity-80" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)"
				onclick={() => navigator.clipboard?.writeText(caddyfile)}>
				<span class="material-symbols-outlined" style="font-size: 18px">content_copy</span>Copy
			</button>
		</div>
		<pre class="overflow-auto p-5 font-mono text-xs leading-6" style="background-color: var(--inverse-surface); color: var(--inverse-on-surface)">{caddyfile}</pre>
	</section>
</AppShell>
