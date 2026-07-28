<script lang="ts">
	import type { TabType, SystemStats } from '$lib/types';

	interface Props {
		activeTab: TabType;
		containersCount: number;
		volumesCount: number;
		systemStats: SystemStats;
		pingMs: number | null;
		onTabSelect: (tab: TabType) => void;
	}

	let { activeTab, containersCount, volumesCount, systemStats, pingMs, onTabSelect }: Props = $props();
</script>

<aside class="w-64 border-r border-neutral-800 bg-neutral-900/60 backdrop-blur-md flex flex-col justify-between p-4 shrink-0">
	<div>
		<!-- Brand Header -->
		<div class="flex items-center gap-3 px-3 py-3 mb-6 border-b border-neutral-800/80">
			<div class="h-9 w-9 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400 font-mono font-bold text-lg shadow-sm shadow-emerald-950">
				>_
			</div>
			<div>
				<h1 class="font-bold text-base tracking-tight text-neutral-100">DevPanel</h1>
				<p class="text-xs text-neutral-400 font-mono">{systemStats.os || 'Linux OS'} {systemStats.arch ? `(${systemStats.arch})` : ''}</p>
			</div>
		</div>

		<!-- Nav Items -->
		<nav class="space-y-1">
			<button
				onclick={() => onTabSelect('overview')}
				class="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all {activeTab === 'overview'
					? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
					: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"/></svg>
				Overview
			</button>

			<button
				onclick={() => onTabSelect('containers')}
				class="w-full flex items-center justify-between px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all {activeTab === 'containers'
					? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
					: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
			>
				<div class="flex items-center gap-3">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
					Containers
				</div>
				<span class="text-xs px-2 py-0.5 rounded-full bg-neutral-800 text-neutral-300 font-mono">{containersCount}</span>
			</button>

			<button
				onclick={() => onTabSelect('volumes')}
				class="w-full flex items-center justify-between px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all {activeTab === 'volumes'
					? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
					: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
			>
				<div class="flex items-center gap-3">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
					Volumes
				</div>
				<span class="text-xs px-2 py-0.5 rounded-full bg-neutral-800 text-neutral-300 font-mono">{volumesCount}</span>
			</button>

			<button
				onclick={() => onTabSelect('blueprints')}
				class="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all {activeTab === 'blueprints'
					? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
					: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
				Blueprints
			</button>

			<button
				onclick={() => onTabSelect('settings')}
				class="w-full flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all {activeTab === 'settings'
					? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 shadow-sm'
					: 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-800/50'}"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>
				Settings
			</button>
		</nav>
	</div>

	<!-- Status & Webpage Ping Footer -->
	<div class="border-t border-neutral-800/80 pt-4 px-2 space-y-2">
		<!-- Webpage Ping RTT Indicator -->
		<div class="flex items-center justify-between text-xs text-neutral-400">
			<span>Webpage Latency</span>
			<span class="inline-flex items-center gap-1.5 font-mono text-xs {pingMs !== null ? 'text-emerald-400' : 'text-rose-400'}">
				<span class="w-2 h-2 rounded-full {pingMs !== null ? 'bg-emerald-400 animate-pulse' : 'bg-rose-500'}"></span>
				{pingMs !== null ? `${pingMs} ms` : 'Offline'}
			</span>
		</div>
		<div class="flex items-center justify-between text-xs text-neutral-400">
			<span>Scale-to-Zero</span>
			<span class="inline-flex items-center gap-1.5 text-emerald-400 font-mono">Active</span>
		</div>
		<div class="flex items-center justify-between text-xs text-neutral-500 font-mono">
			<span>Systemd Socket</span>
			<span>LISTEN_FDS=1</span>
		</div>
	</div>
</aside>
