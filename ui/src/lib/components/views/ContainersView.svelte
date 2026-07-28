<script lang="ts">
	import type { Container, Volume, BlueprintItem, DeleteTarget, ErrorModalState, LogStreamState } from '$lib/types';
	import DropdownMenu from '$lib/components/DropdownMenu.svelte';

	interface Props {
		containers: Container[];
		loading: boolean;
		actionLoading: string | null;
		onToggleStatus: (container: Container) => void;
		onOpenLogs: (container: Container) => void;
		onPromptDelete: (container: Container) => void;
	}

	let { containers, loading, actionLoading, onToggleStatus, onOpenLogs, onPromptDelete }: Props = $props();

	function getStatusBadgeClass(status: string) {
		switch (status) {
			case 'running':
			case 'live':
				return 'bg-green-100 text-green-700 border-green-200';
			case 'deploying':
			case 'restarting':
				return 'bg-yellow-100 text-yellow-700 border-yellow-200 animate-pulse';
			case 'failed':
			case 'error':
				return 'bg-red-100 text-red-700 border-red-200';
			default:
				return 'bg-gray-100 text-gray-700 border-gray-200';
		}
	}
</script>

<div class="p-6 md:p-10 max-w-7xl mx-auto w-full">
	<div class="mb-8">
		<h1 class="text-2xl font-semibold text-gray-900 tracking-tight">Docker Containers</h1>
		<p class="text-gray-500 mt-1">Manage underlying container instances.</p>
	</div>

	<div class="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
		<div class="overflow-x-auto">
			<table class="w-full text-left text-sm">
				<thead class="bg-gray-50 border-b border-gray-200 text-gray-600">
					<tr>
						<th class="px-6 py-4 font-medium">Container Name</th>
						<th class="px-6 py-4 font-medium">Image</th>
						<th class="px-6 py-4 font-medium">Status</th>
						<th class="px-6 py-4 font-medium">Ports</th>
						<th class="px-6 py-4 font-medium text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100">
					{#if containers.length === 0}
						<tr>
							<td colspan="5" class="px-6 py-8 text-center text-gray-500">
								No active Docker containers found on system.
							</td>
						</tr>
					{:else}
						{#each containers as container}
							<tr class="hover:bg-gray-50 transition-colors">
								<td class="px-6 py-4 font-medium text-gray-900 flex items-center gap-3">
									<svg class="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
									<span>{container.name}</span>
								</td>
								<td class="px-6 py-4 font-mono text-gray-500">{container.image}</td>
								<td class="px-6 py-4">
									<span class="px-2.5 py-0.5 rounded-full text-xs font-medium border uppercase tracking-wider {getStatusBadgeClass(container.status)}">
										{container.status}
									</span>
								</td>
								<td class="px-6 py-4 text-gray-500 font-mono text-xs">{container.port || '-'}</td>
								<td class="px-6 py-4 text-right flex justify-end gap-2">
									<button
										type="button"
										onclick={() => onToggleStatus(container)}
										disabled={actionLoading === container.id}
										class="p-1.5 text-gray-400 hover:text-green-600 hover:bg-green-50 rounded transition-colors disabled:opacity-50"
										title={container.status === 'running' ? 'Stop Container' : 'Start Container'}
									>
										{#if container.status === 'running'}
											<svg class="w-4 h-4 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"/></svg>
										{:else}
											<svg class="w-4 h-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
										{/if}
									</button>
									<button
										type="button"
										onclick={() => onOpenLogs(container)}
										class="p-1.5 text-gray-400 hover:text-gray-900 hover:bg-gray-100 rounded transition-colors"
										title="View Terminal Logs"
									>
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
									</button>
									<button
										type="button"
										onclick={() => onPromptDelete(container)}
										class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
										title="Delete Container"
									>
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
									</button>
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>
	</div>
</div>
