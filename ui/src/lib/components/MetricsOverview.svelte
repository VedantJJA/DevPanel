<script lang="ts">
	import type { Container, SystemStats } from '$lib/types';

	interface Props {
		containers: Container[];
		systemStats: SystemStats;
	}

	let { containers, systemStats }: Props = $props();

	let activeCount = $derived(containers.filter(c => c.status === 'running').length);
	
	// Sum of container CPU loads where 2 cores @ 100% load = 200%
	let containerCpuSum = $derived(
		Math.round(containers.filter(c => c.status === 'running').reduce((acc, c) => acc + (c.cpuPercent || 0), 0) * 10) / 10
	);

	// Display exact multi-core CPU percentage (where N cores @ 100% = N * 100%)
	let displayCpuPercent = $derived(
		systemStats.cpuPercent !== undefined && systemStats.cpuPercent > 0
			? systemStats.cpuPercent
			: containerCpuSum
	);

	let maxCpuPercent = $derived(Math.max(100, systemStats.cpus * 100));
	let cpuBarWidth = $derived(
		maxCpuPercent > 0 ? Math.min(100, Math.round((displayCpuPercent / maxCpuPercent) * 100)) : 0
	);
</script>

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

	<!-- Multi-Core CPU Load (e.g. 2 cores at 100% = 200%) -->
	<div class="p-5 rounded-2xl bg-neutral-900/70 border border-neutral-800 flex flex-col justify-between shadow-sm">
		<div class="flex items-center justify-between text-neutral-400 text-xs font-medium">
			<span>CPU Core Count & Load</span>
			<span class="p-1.5 rounded-lg bg-sky-500/10 text-sky-400">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M3 9h2m-2 6h2m14-6h2m-2 6h2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/></svg>
			</span>
		</div>
		<div class="mt-3 flex items-baseline gap-2">
			<span class="text-3xl font-bold text-neutral-100 font-mono">{systemStats.cpus} <span class="text-sm font-normal text-neutral-400">{systemStats.cpus === 1 ? 'Core' : 'Cores'}</span></span>
			<span class="text-xs text-sky-400 font-mono">({displayCpuPercent}% load)</span>
		</div>
		<div class="mt-3 w-full bg-neutral-800 h-1.5 rounded-full overflow-hidden">
			<div class="bg-sky-500 h-full transition-all duration-500" style="width: {cpuBarWidth}%"></div>
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
