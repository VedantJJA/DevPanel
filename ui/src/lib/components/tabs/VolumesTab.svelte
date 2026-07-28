<script lang="ts">
	import type { Volume } from '$lib/types';

	interface Props {
		volumes: Volume[];
		loading: boolean;
		onPromptDelete: (volume: Volume) => void;
	}

	let { volumes, loading, onPromptDelete }: Props = $props();
</script>

<section class="space-y-4">
	<div class="flex items-center justify-between">
		<h3 class="text-lg font-bold text-neutral-100">Docker Volumes</h3>
		<span class="text-xs text-neutral-400 font-mono">{volumes.length} Volumes found</span>
	</div>

	<div class="rounded-2xl border border-neutral-800 bg-neutral-900/60 overflow-hidden shadow-xl">
		<div class="overflow-x-auto">
			<table class="w-full text-left border-collapse">
				<thead>
					<tr class="border-b border-neutral-800 bg-neutral-900/80 text-xs font-medium text-neutral-400 uppercase tracking-wider">
						<th class="py-4 px-6">Volume Name</th>
						<th class="py-4 px-4">Driver</th>
						<th class="py-4 px-6">Mountpoint Path</th>
						<th class="py-4 px-4">Scope</th>
						<th class="py-4 px-6 text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-neutral-800/60 text-sm">
					{#if loading && volumes.length === 0}
						<tr>
							<td colspan="5" class="py-12 text-center text-neutral-500">
								Fetching volumes from Docker engine...
							</td>
						</tr>
					{:else if volumes.length === 0}
						<tr>
							<td colspan="5" class="py-12 text-center text-neutral-500">
								No Docker volumes found on server.
							</td>
						</tr>
					{:else}
						{#each volumes as volume}
							<tr class="hover:bg-neutral-800/30 transition-colors">
								<td class="py-4 px-6 font-medium text-neutral-100 font-mono text-xs">
									{volume.Name}
								</td>
								<td class="py-4 px-4 text-xs font-mono text-emerald-400">
									{volume.Driver || 'local'}
								</td>
								<td class="py-4 px-6 text-xs font-mono text-neutral-400">
									{volume.Mountpoint}
								</td>
								<td class="py-4 px-4 text-xs font-mono text-neutral-400">
									{volume.Scope || 'local'}
								</td>
								<td class="py-4 px-6 text-right">
									<button
										onclick={() => onPromptDelete(volume)}
										title="Delete Volume"
										class="px-3 py-1.5 rounded-lg bg-neutral-800 hover:bg-rose-600/30 text-rose-400 text-xs font-semibold border border-neutral-700/60 hover:border-rose-500/30 transition-all"
									>
										Delete
									</button>
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>
	</div>
</section>
