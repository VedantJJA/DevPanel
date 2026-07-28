<script lang="ts">
	import type { BlueprintItem } from '$lib/types';

	interface Props {
		blueprints: BlueprintItem[];
		loading: boolean;
		actionLoading: string | null;
		onDeployBlueprint: (bp: BlueprintItem) => void;
		onPromptDeleteBlueprint: (bp: BlueprintItem) => void;
	}

	let { blueprints, loading, actionLoading, onDeployBlueprint, onPromptDeleteBlueprint }: Props = $props();
</script>

<section class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h3 class="text-lg font-bold text-neutral-100">Application Blueprints</h3>
			<p class="text-xs text-neutral-400">Manage multi-tier monorepos and stack blueprints</p>
		</div>

		<a
			href="/blueprints/new"
			class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs flex items-center gap-2 transition-all shadow-md shadow-emerald-950/60"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
			+ Add Blueprint App
		</a>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
		<!-- Built-in Monorepo Sample Card if empty -->
		{#if blueprints.length === 0}
			<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-4 flex flex-col justify-between hover:border-neutral-700 transition-all shadow-lg">
				<div class="space-y-3">
					<div class="flex items-center justify-between">
						<span class="px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">Monorepo Sample</span>
						<span class="text-xs font-mono text-neutral-500">v1.0</span>
					</div>
					<h4 class="font-bold text-neutral-100 text-base">my-monorepo-startup</h4>
					<p class="text-xs text-neutral-400 line-clamp-2">
						Multi-tier PaaS architecture (Svelte frontend, Go API backend, PostgreSQL 15 database).
					</p>
					<div class="pt-2 flex flex-wrap gap-2 text-xs font-mono">
						<span class="px-2 py-0.5 rounded bg-neutral-800 text-neutral-300">ui/ (node)</span>
						<span class="px-2 py-0.5 rounded bg-neutral-800 text-neutral-300">api/ (dockerfile)</span>
						<span class="px-2 py-0.5 rounded bg-neutral-800 text-neutral-300">postgres:15</span>
					</div>
				</div>

				<div class="pt-4 border-t border-neutral-800/80 flex items-center justify-between">
					<a href="/blueprints/new" class="text-xs text-emerald-400 hover:underline font-mono">Index Blueprint →</a>
					<a href="/blueprints/new" class="px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold">Deploy</a>
				</div>
			</div>
		{/if}

		{#each blueprints as bp (bp.id)}
			<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-4 flex flex-col justify-between hover:border-neutral-700 transition-all shadow-lg">
				<div class="space-y-3">
					<div class="flex items-center justify-between">
						<span class="px-2.5 py-1 rounded-full text-xs font-semibold capitalize border {bp.status === 'active'
							? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
							: 'bg-neutral-800 text-neutral-400 border-neutral-700'}">
							{bp.status}
						</span>
						<span class="text-xs font-mono text-neutral-500">{bp.createdAt}</span>
					</div>

					<h4 class="font-bold text-neutral-100 text-base">{bp.name}</h4>
					<p class="text-xs text-neutral-400 font-mono truncate" title={bp.repo_url}>
						{bp.repo_url}
					</p>
				</div>

				<div class="pt-4 border-t border-neutral-800/80 flex items-center justify-between gap-3">
					<button
						onclick={() => onPromptDeleteBlueprint(bp)}
						class="p-1.5 rounded-lg bg-neutral-800 hover:bg-rose-600/30 text-neutral-400 hover:text-rose-400 border border-neutral-700/60 transition-all"
						title="Delete Blueprint"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
					</button>

					<button
						onclick={() => onDeployBlueprint(bp)}
						disabled={actionLoading === bp.id}
						class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold transition-all disabled:opacity-50"
					>
						{actionLoading === bp.id ? 'Deploying...' : 'Deploy Application'}
					</button>
				</div>
			</div>
		{/each}
	</div>
</section>
