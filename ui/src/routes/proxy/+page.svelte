<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import type { SystemStats } from '$lib/types';

	let systemStats = $state<SystemStats>({ totalContainers: 0, activeContainers: 0, stoppedContainers: 0, totalMemMb: 0, usedMemMb: 0, memPercent: 0, cpus: 1 });

	let routes = $state<{ host: string; upstream: string; tls: string; status: string; requests: string }[]>([]);
	let debugLogs = $state<any[]>([]);
	let loading = $state(true);
	let timer: any;

	const caddyfile = `# DevPanel managed Caddyfile
# Auto-generated from routing config

:80 {
    reverse_proxy localhost:8090
}`;

	async function fetchData() {
		try {
			const sRes = await fetch('/api/system/stats');
			if (sRes.ok) { const s = await sRes.json(); systemStats = { ...systemStats, ...s }; }
			

			try {
				const dbgRes = await fetch('/api/debug/routes');
				if (dbgRes.ok) { const d = await dbgRes.json(); debugLogs = (d.routes || []).reverse(); }
			} catch { debugLogs = []; }
		} catch (e) { console.error(e); } finally { loading = false; }
	}

	onMount(() => {
		fetchData();
		timer = setInterval(fetchData, 3000);
	});

	onDestroy(() => {
		if (timer) clearInterval(timer);
	});
</script>

<AppShell {systemStats}>
	<!-- Header -->
	<div class="mb-6">
		<div class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<h1 class="text-[28px] font-bold leading-tight lg:text-[32px]" style="color: var(--on-surface)">Caddy & Internal Reverse Proxy</h1>
				<p class="mt-1" style="color: var(--on-surface-variant)">Live hostnames, upstream target containers, and real-time routing debugger.</p>
			</div>
			<div class="flex flex-wrap gap-2">
				<button onclick={fetchData} class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
					<span class="material-symbols-outlined" style="font-size: 20px">restart_alt</span>Reload Config
				</button>
			</div>
		</div>
	</div>

	<!-- Stat Cards -->
	<div class="mb-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Active Routes</span>
			<span class="text-2xl font-bold" style="color: var(--primary)">{debugLogs.length} logged</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Certificates</span>
			<span class="text-2xl font-bold" style="color: var(--success)">Auto-TLS</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">API Rerouting</span>
			<span class="text-2xl font-bold" style="color: var(--primary)">Active</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">Frontend → Backend</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Proxy Engine</span>
			<span class="text-2xl font-bold">DevPanel Proxy</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">Render-style Slugs</span>
		</div>
	</div>

	<!-- LIVE ROUTING DEBUG PANEL -->
	<section class="card-surface mb-6 overflow-hidden">
		<div class="flex items-center justify-between border-b px-5 py-4" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
			<div class="flex items-center gap-2">
				<span class="material-symbols-outlined text-blue-400" style="font-size: 24px">bug_report</span>
				<h2 class="font-bold text-base" style="color: var(--on-surface)">Live Routing Debug Panel</h2>
			</div>
			<span class="rounded px-2.5 py-1 text-xs font-mono font-medium bg-blue-500/10 text-blue-400 border border-blue-500/20">
				Real-time Logs (Auto-refresh 3s)
			</span>
		</div>

		{#if debugLogs.length === 0}
			<div class="flex flex-col items-center py-10 text-center">
				<span class="material-symbols-outlined mb-2" style="font-size: 36px; color: var(--outline)">alt_route</span>
				<p class="text-sm" style="color: var(--on-surface-variant)">No request routes captured yet.</p>
				<p class="mt-1 text-xs" style="color: var(--on-surface-variant)">Requests to hosted application domains will appear here instantly.</p>

			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-left font-mono text-xs">
					<thead>
						<tr class="border-b" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
							<th class="px-4 py-2.5" style="color: var(--on-surface-variant)">Time</th>
							<th class="px-4 py-2.5" style="color: var(--on-surface-variant)">Method</th>
							<th class="px-4 py-2.5" style="color: var(--on-surface-variant)">Requested Path</th>
							<th class="px-4 py-2.5" style="color: var(--on-surface-variant)">Resolved Slug</th>
							<th class="px-4 py-2.5" style="color: var(--on-surface-variant)">Target Service</th>
							<th class="px-4 py-2.5" style="color: var(--on-surface-variant)">Type</th>
							<th class="px-4 py-2.5" style="color: var(--on-surface-variant)">Upstream Target</th>
						</tr>
					</thead>
					<tbody class="divide-y" style="border-color: var(--outline-variant)">
						{#each debugLogs as log}
							<tr class="hover:bg-white/5 transition-colors">
								<td class="px-4 py-2.5 text-gray-400 whitespace-nowrap">{log.timestamp ? log.timestamp.split('T')[1]?.replace('Z', '') : '—'}</td>
								<td class="px-4 py-2.5 font-bold whitespace-nowrap">
									<span class={log.method === 'GET' ? 'text-green-400' : 'text-blue-400'}>{log.method}</span>
								</td>
								<td class="px-4 py-2.5 text-gray-200 truncate max-w-[220px]" title={log.path}>{log.path}</td>
								<td class="px-4 py-2.5 text-purple-300 font-semibold">{log.resolved_slug || '—'}</td>
								<td class="px-4 py-2.5 font-semibold text-yellow-300">
									{log.service_name}
									{#if log.is_api_reroute}
										<span class="ml-1.5 inline-block rounded bg-emerald-500/20 px-1.5 py-0.5 text-[10px] text-emerald-400 border border-emerald-500/30">API Rerouted</span>
									{/if}
								</td>
								<td class="px-4 py-2.5">
									<span class="rounded px-1.5 py-0.5 text-[10px] uppercase font-bold" style="background-color: var(--surface-high); color: var(--on-surface-variant)">
										{log.service_type || 'web'}
									</span>
								</td>
								<td class="px-4 py-2.5 text-emerald-400 font-bold whitespace-nowrap">{log.upstream_url}</td>
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
Shell>
