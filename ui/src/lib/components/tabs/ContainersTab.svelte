<script lang="ts">
	import type { Container } from '$lib/types';

	interface Props {
		containers: Container[];
		loading: boolean;
		actionLoading: string | null;
		onToggleStatus: (container: Container) => void;
		onOpenLogs: (container: Container) => void;
		onPromptDelete: (container: Container) => void;
	}

	let { containers, loading, actionLoading, onToggleStatus, onOpenLogs, onPromptDelete }: Props = $props();

	let searchQuery = $state('');
	let statusFilter = $state('all');

	let filteredContainers = $derived(
		containers.filter((c) => {
			const matchesSearch =
				c.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				c.image.toLowerCase().includes(searchQuery.toLowerCase());
			const matchesStatus = statusFilter === 'all' || c.status === statusFilter;
			return matchesSearch && matchesStatus;
		})
	);
</script>

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
											onclick={() => onOpenLogs(container)}
											class="px-2.5 py-1.5 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-300 text-xs font-medium border border-neutral-700/60 transition-all"
										>
											Logs
										</button>

										<button
											onclick={() => onToggleStatus(container)}
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
											onclick={() => onPromptDelete(container)}
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
