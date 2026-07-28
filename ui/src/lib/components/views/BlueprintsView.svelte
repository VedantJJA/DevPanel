<script lang="ts">
	import type { BlueprintItem } from '$lib/types';

	interface Props {
		blueprints: BlueprintItem[];
		loading: boolean;
		actionLoading: string | null;
		onDeployBlueprint: (bp: BlueprintItem) => void;
		onPromptDeleteBlueprint: (bp: BlueprintItem) => void;
		onCreateBlueprint: () => void;
	}

	let { blueprints, loading, actionLoading, onDeployBlueprint, onPromptDeleteBlueprint, onCreateBlueprint }: Props = $props();
</script>

<div class="p-6 md:p-10 max-w-7xl mx-auto w-full">
	<div class="flex flex-col sm:flex-row sm:items-center justify-between mb-8 gap-4">
		<div>
			<h1 class="text-2xl font-semibold text-gray-900 tracking-tight">Blueprints</h1>
			<p class="text-gray-500 mt-1">Reusable infrastructure templates.</p>
		</div>
		<button
			type="button"
			onclick={onCreateBlueprint}
			class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors flex items-center justify-center gap-2 shadow-sm"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
			<span>Create Blueprint</span>
		</button>
	</div>

	{#if blueprints.length === 0}
		<div class="bg-white border border-gray-200 rounded-xl p-12 text-center text-gray-500 shadow-sm">
			<svg class="w-12 h-12 text-gray-300 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
			<h3 class="text-base font-semibold text-gray-900 mb-1">No Blueprints Configured</h3>
			<p class="text-sm text-gray-500 max-w-md mx-auto mb-6">Create or deploy application blueprints using devpanel.yaml files.</p>
			<button
				type="button"
				onclick={onCreateBlueprint}
				class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium text-sm transition-colors inline-flex items-center gap-2"
			>
				+ Deploy New Service
			</button>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
			{#each blueprints as bp}
				<div class="bg-white border border-gray-200 rounded-xl p-6 shadow-sm flex flex-col hover:border-blue-300 hover:shadow-md transition-all group">
					<div class="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center mb-4 border border-blue-100 group-hover:bg-blue-600 transition-colors">
						<svg class="w-5 h-5 text-blue-600 group-hover:text-white transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
					</div>
					<h3 class="text-lg font-semibold text-gray-900 mb-1">{bp.name}</h3>
					<p class="text-sm text-gray-500 mb-4 flex-1">
						{bp.serviceCount || 1} Service(s) configured in blueprint template.
					</p>
					<div class="pt-4 border-t border-gray-100 flex items-center justify-between text-sm">
						<span class="text-gray-400 flex items-center gap-1.5 font-mono text-xs truncate max-w-[160px]">
							<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
							{bp.repo_url}
						</span>
						<button
							type="button"
							onclick={() => onDeployBlueprint(bp)}
							disabled={actionLoading === bp.id}
							class="text-blue-600 font-medium hover:text-blue-700 disabled:opacity-50"
						>
							{actionLoading === bp.id ? 'Deploying...' : 'Deploy'}
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
