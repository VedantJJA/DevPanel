<script lang="ts">
	import type { TabType } from '$lib/types';

	interface Props {
		activeTab: TabType;
		loading: boolean;
		actionLoading: string | null;
		pingMs: number | null;
		onRefresh: () => void;
		onStartAll: () => void;
		onStopAll: () => void;
	}

	let { activeTab, loading, actionLoading, pingMs, onRefresh, onStartAll, onStopAll }: Props = $props();
</script>

<header class="border-b border-neutral-800 bg-neutral-900/40 px-8 py-5 flex items-center justify-between gap-4 sticky top-0 backdrop-blur-md z-10">
	<div>
		<h2 class="text-xl font-bold tracking-tight text-neutral-100 capitalize">{activeTab} Dashboard</h2>
		<p class="text-xs text-neutral-400 mt-0.5">Real-time system telemetry and Docker resource controls</p>
	</div>

	<!-- Action Bar & Ping Badge -->
	<div class="flex items-center gap-3">
		<!-- Webpage Ping Badge -->
		<div class="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-neutral-900 border border-neutral-800 text-xs font-mono text-neutral-300">
			<span class="w-2 h-2 rounded-full {pingMs !== null ? 'bg-emerald-400 animate-pulse' : 'bg-rose-500'}"></span>
			<span>Ping: <strong class="text-neutral-100">{pingMs !== null ? `${pingMs} ms` : 'Disconnected'}</strong></span>
		</div>

		<button
			onclick={onRefresh}
			disabled={loading}
			class="px-3.5 py-2 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-semibold flex items-center gap-2 border border-neutral-700/60 transition-all disabled:opacity-50"
		>
			<svg class="w-3.5 h-3.5 {loading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
			Refresh
		</button>

		{#if activeTab === 'containers' || activeTab === 'overview'}
			<button
				onclick={onStartAll}
				disabled={actionLoading !== null}
				class="px-3.5 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold flex items-center gap-2 transition-all shadow-sm shadow-emerald-950 disabled:opacity-50"
			>
				{#if actionLoading === 'start-all'}
					<svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
				{:else}
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
				{/if}
				Start All
			</button>

			<button
				onclick={onStopAll}
				disabled={actionLoading !== null}
				class="px-3.5 py-2 rounded-lg bg-rose-600/90 hover:bg-rose-500 text-white text-xs font-semibold flex items-center gap-2 transition-all shadow-sm shadow-rose-950 disabled:opacity-50"
			>
				{#if actionLoading === 'stop-all'}
					<svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
				{:else}
					<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"/></svg>
				{/if}
				Stop All
			</button>
		{/if}
	</div>
</header>
