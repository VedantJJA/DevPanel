<script lang="ts">
	import { onMount } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import type { SystemStats } from '$lib/types';

	let systemStats = $state<SystemStats>({ totalContainers: 0, activeContainers: 0, stoppedContainers: 0, totalMemMb: 0, usedMemMb: 0, memPercent: 0, cpus: 1 });
	let pingMs = $state<number | null>(null);

	const cpuSeries = [18, 22, 19, 31, 44, 38, 29, 26, 34, 52, 61, 47, 39, 33, 28, 24, 27, 35, 42, 30, 25, 22, 20, 24];
	const memSeries = [40, 42, 41, 45, 48, 52, 55, 54, 53, 58, 62, 60, 59, 57, 56, 55, 54, 56, 58, 57, 55, 54, 53, 58];

	function sparklinePoints(data: number[]): string {
		const max = Math.max(...data);
		return data.map((v, i) => `${(i / (data.length - 1)) * 100},${100 - (v / max) * 90}`).join(' ');
	}

	async function fetchData() {
		try {
			const [sRes] = await Promise.all([fetch('/api/system/stats')]);
			if (sRes.ok) { const s = await sRes.json(); systemStats = { ...systemStats, ...s }; }
		} catch (e) { console.error(e); }
	}

	async function measurePing() {
		const start = performance.now();
		try { const r = await fetch('/healthz', { cache: 'no-store' }); if (r.ok) pingMs = Math.round(performance.now() - start); } catch { pingMs = null; }
	}

	onMount(async () => { await fetchData(); await measurePing(); });

	const cpuColor = 'oklch(0.4845 0.2111 261.13)';
	const memColor = 'oklch(0.4822 0.077 254.45)';
</script>

<AppShell {systemStats} {pingMs}>
	<!-- Header -->
	<div class="mb-6">
		<div class="flex flex-wrap items-end justify-between gap-4">
			<div>
				<h1 class="text-[28px] font-bold leading-tight lg:text-[32px]" style="color: var(--on-surface)">Metrics</h1>
				<p class="mt-1" style="color: var(--on-surface-variant)">System resource overview · sampled live.</p>
			</div>
			<div class="flex flex-wrap gap-2">
				<button onclick={fetchData} class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
					<span class="material-symbols-outlined" style="font-size: 20px">schedule</span>Live
				</button>
				<button class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
					<span class="material-symbols-outlined" style="font-size: 20px">download</span>Export CSV
				</button>
			</div>
		</div>
	</div>

	<!-- Stat Cards -->
	<div class="mb-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Avg CPU</span>
			<span class="text-2xl font-bold" style="color: var(--primary)">{(systemStats.cpuPercent ?? 0).toFixed(1)}%</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">{systemStats.cpus} vCPU</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Memory used</span>
			<span class="text-2xl font-bold">{(systemStats.usedMemMb / 1024).toFixed(1)} GB</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">of {(systemStats.totalMemMb / 1024).toFixed(0)} GB</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Containers</span>
			<span class="text-2xl font-bold">{systemStats.activeContainers}/{systemStats.totalContainers}</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">running</span>
		</div>
		<div class="card-surface flex flex-col gap-1 p-5">
			<span class="label-caps" style="color: var(--on-surface-variant)">Latency</span>
			<span class="text-2xl font-bold" style="color: {pingMs !== null ? 'var(--success)' : 'var(--error)'}">{pingMs !== null ? `${pingMs}ms` : 'N/A'}</span>
			<span class="text-xs" style="color: var(--on-surface-variant)">to backend</span>
		</div>
	</div>

	<!-- Sparkline Charts -->
	<div class="mb-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
		<section class="card-surface p-5">
			<h2 class="mb-1 font-bold">CPU Utilisation</h2>
			<p class="mb-3 text-xs" style="color: var(--on-surface-variant)">% of {systemStats.cpus} vCPU (sample window)</p>
			<svg viewBox="0 0 100 100" preserveAspectRatio="none" class="h-40 w-full">
				<polyline points={sparklinePoints(cpuSeries)} fill="none" stroke={cpuColor} stroke-width="1.5" vector-effect="non-scaling-stroke" />
				<polyline points={`0,100 ${sparklinePoints(cpuSeries)} 100,100`} fill={cpuColor} opacity="0.12" stroke="none" />
			</svg>
		</section>
		<section class="card-surface p-5">
			<h2 class="mb-1 font-bold">Memory Utilisation</h2>
			<p class="mb-3 text-xs" style="color: var(--on-surface-variant)">% of {(systemStats.totalMemMb / 1024).toFixed(0)} GB RAM</p>
			<svg viewBox="0 0 100 100" preserveAspectRatio="none" class="h-40 w-full">
				<polyline points={sparklinePoints(memSeries)} fill="none" stroke={memColor} stroke-width="1.5" vector-effect="non-scaling-stroke" />
				<polyline points={`0,100 ${sparklinePoints(memSeries)} 100,100`} fill={memColor} opacity="0.12" stroke="none" />
			</svg>
		</section>
	</div>

	<!-- Resource bars -->
	<section class="card-surface p-5">
		<h2 class="mb-4 font-bold">Resource breakdown</h2>
		{#each [
			{ label: 'CPU', value: systemStats.cpuPercent ?? 0, detail: `${(systemStats.cpuPercent ?? 0).toFixed(1)}% of ${systemStats.cpus} cores` },
			{ label: 'Memory', value: systemStats.memPercent ?? 0, detail: `${(systemStats.usedMemMb/1024).toFixed(1)} of ${(systemStats.totalMemMb/1024).toFixed(0)} GB` },
		] as r}
			<div class="mb-4 last:mb-0">
				<div class="mb-1 flex justify-between text-sm">
					<span class="font-medium">{r.label}</span>
					<span style="color: var(--on-surface-variant)">{r.detail}</span>
				</div>
				<div class="h-2 w-full rounded-full" style="background-color: var(--surface-high)">
					<div class="h-2 rounded-full transition-all" style="width: {r.value}%; background-color: var(--primary)"></div>
				</div>
			</div>
		{/each}
	</section>
</AppShell>
