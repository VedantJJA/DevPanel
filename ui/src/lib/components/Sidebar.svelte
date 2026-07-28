<script lang="ts">
	import type { TabType, SystemStats, ProjectDetail } from '$lib/types';
	import DropdownMenu from './DropdownMenu.svelte';

	interface Props {
		appView: string;
		selectedProject: ProjectDetail | null;
		selectedServiceIdx: number;
		systemStats: SystemStats;
		pingMs: number | null;
		onNavigate: (view: string) => void;
		onSelectServiceTab: (tab: string) => void;
		activeServiceTab: string;
		onBackToDashboard: () => void;
	}

	let {
		appView,
		selectedProject,
		selectedServiceIdx,
		systemStats,
		pingMs,
		onNavigate,
		onSelectServiceTab,
		activeServiceTab,
		onBackToDashboard
	}: Props = $props();

	const globalNav = [
		{ id: 'dashboard', label: 'Overview', icon: 'dashboard' },
		{ id: 'containers', label: 'Containers', icon: 'box' },
		{ id: 'blueprints', label: 'Blueprints', icon: 'layers' },
		{ id: 'workspaces', label: 'Workspaces', icon: 'briefcase' },
		{ id: 'settings', label: 'Settings', icon: 'settings' }
	];

	const serviceNav = [
		{ id: 'events', label: 'Events', icon: 'activity' },
		{ id: 'logs', label: 'Logs', icon: 'terminal' },
		{ id: 'env', label: 'Environment', icon: 'database' },
		{ id: 'domains', label: 'Custom Domains', icon: 'globe' },
		{ id: 'metrics', label: 'Metrics', icon: 'activity' },
		{ id: 'settings', label: 'Settings', icon: 'settings' }
	];
</script>

<aside class="w-64 border-r border-gray-200 bg-white hidden md:flex flex-col shrink-0 z-20 h-screen">
	<!-- Brand Header -->
	<div class="h-16 flex items-center px-6 border-b border-gray-200 shrink-0">
		<button type="button" class="flex items-center gap-2.5 text-gray-900 font-bold text-lg tracking-tight cursor-pointer" onclick={onBackToDashboard}>
			<div class="w-8 h-8 bg-blue-600 rounded-xl flex items-center justify-center shadow-sm text-white">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/></svg>
			</div>
			<span>DevPanel</span>
		</button>
	</div>

	<!-- Sidebar Content Navigation -->
	<nav class="flex-1 p-4 space-y-1 overflow-y-auto">
		{#if selectedProject && selectedProject.services[selectedServiceIdx]}
			{@const currentSvc = selectedProject.services[selectedServiceIdx]}
			<button
				type="button"
				onclick={onBackToDashboard}
				class="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium text-gray-500 hover:text-gray-900 hover:bg-gray-100 transition-colors mb-4"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/></svg>
				Back to System
			</button>
			<div class="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2 mt-4 truncate">
				{currentSvc.name}
			</div>
			{#each serviceNav as item}
				<button
					type="button"
					onclick={() => onSelectServiceTab(item.id)}
					class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors {activeServiceTab === item.id
						? 'bg-blue-50 text-blue-700'
						: 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'}"
				>
					{#if item.icon === 'activity'}
						<svg class="w-4 h-4 {activeServiceTab === item.id ? 'text-blue-600' : 'text-gray-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
					{:else if item.icon === 'terminal'}
						<svg class="w-4 h-4 {activeServiceTab === item.id ? 'text-blue-600' : 'text-gray-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
					{:else if item.icon === 'database'}
						<svg class="w-4 h-4 {activeServiceTab === item.id ? 'text-blue-600' : 'text-gray-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"/></svg>
					{:else if item.icon === 'globe'}
						<svg class="w-4 h-4 {activeServiceTab === item.id ? 'text-blue-600' : 'text-gray-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0zM3.6 9h16.8M3.6 15h16.8"/></svg>
					{:else if item.icon === 'settings'}
						<svg class="w-4 h-4 {activeServiceTab === item.id ? 'text-blue-600' : 'text-gray-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/></svg>
					{/if}
					<span>{item.label}</span>
				</button>
			{/each}
		{:else}
			<div class="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">
				System
			</div>
			{#each globalNav as item}
				<button
					type="button"
					onclick={() => onNavigate(item.id)}
					class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors {appView === item.id
						? 'bg-blue-50 text-blue-700'
						: 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'}"
				>
					{#if item.icon === 'dashboard'}
						<svg class="w-4 h-4 {appView === item.id ? 'text-blue-600' : 'text-gray-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"/></svg>
					{:else if item.icon === 'box'}
						<svg class="w-4 h-4 {appView === item.id ? 'text-blue-600' : 'text-gray-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
					{:else if item.icon === 'layers'}
						<svg class="w-4 h-4 {appView === item.id ? 'text-blue-600' : 'text-gray-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
					{:else if item.icon === 'briefcase'}
						<svg class="w-4 h-4 {appView === item.id ? 'text-blue-600' : 'text-gray-400'}" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 13.255A23.931 23.931 0 0112 15c-3.183 0-6.22-.62-9-1.745M16 6V4a2 2 0 00-2-2h-4a2 2 0 00-2 2v2m4 6h.01M5 20h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
					{/if}
					<span>{item.label}</span>
				</button>
			{/each}
		{/if}
	</nav>

	<!-- Footer Info -->
	<div class="p-4 border-t border-gray-200 bg-gray-50/50 space-y-3 shrink-0">
		<div class="flex items-center justify-between text-xs text-gray-500">
			<span>Latency</span>
			<span class="inline-flex items-center gap-1.5 font-mono text-xs {pingMs !== null ? 'text-emerald-600 font-medium' : 'text-rose-500'}">
				<span class="w-2 h-2 rounded-full {pingMs !== null ? 'bg-emerald-500 animate-pulse' : 'bg-rose-500'}"></span>
				{pingMs !== null ? `${pingMs} ms` : 'Offline'}
			</span>
		</div>

		<div class="flex items-center gap-3 pt-2 border-t border-gray-200/80">
			<div class="w-8 h-8 rounded-full bg-blue-100 border border-blue-200 flex items-center justify-center text-blue-700 font-bold text-xs shadow-sm">
				DP
			</div>
			<div class="flex-1 min-w-0">
				<p class="text-xs font-semibold text-gray-900 truncate">DevPanel System</p>
				<p class="text-[10px] text-gray-500 truncate">{systemStats.os || 'Linux Runtime'} {systemStats.arch ? `(${systemStats.arch})` : ''}</p>
			</div>
		</div>
	</div>
</aside>
