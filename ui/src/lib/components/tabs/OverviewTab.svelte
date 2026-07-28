<script lang="ts">
	import type { Container, Volume, SystemStats } from '$lib/types';

	interface Props {
		containers: Container[];
		volumes: Volume[];
		systemStats: SystemStats;
	}

	let { containers, volumes, systemStats }: Props = $props();

	let activeCount = $derived(containers.filter(c => c.status === 'running').length);
</script>

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
