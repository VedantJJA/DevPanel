<script lang="ts">
	import type { Container, SystemStats } from '$lib/types';
	import DropdownMenu from '$lib/components/DropdownMenu.svelte';

	interface Props {
		containers: Container[];
		systemStats: SystemStats;
		onSelectService: (service: any) => void;
		onNewService: () => void;
		onToggleStatus: (container: Container) => void;
		onOpenLogs: (container: Container) => void;
		onPromptDelete: (container: Container) => void;
	}

	let {
		containers,
		systemStats,
		onSelectService,
		onNewService,
		onToggleStatus,
		onOpenLogs,
		onPromptDelete
	}: Props = $props();

	let search = $state('');

	const stats = $derived([
		{ label: 'CPU Usage', value: `${systemStats.cpuPercent ?? 0}%`, width: `${Math.min(100, Math.max(0, systemStats.cpuPercent ?? 0))}%`, color: 'bg-blue-500', icon: 'cpu' },
		{ label: 'Memory', value: `${systemStats.usedMemMb || 0} / ${systemStats.totalMemMb || 1024} MB`, width: `${systemStats.memPercent || 0}%`, color: 'bg-indigo-500', icon: 'activity' },
		{ label: 'Containers', value: `${systemStats.activeContainers} Active / ${systemStats.totalContainers}`, width: `${systemStats.totalContainers ? (systemStats.activeContainers / systemStats.totalContainers) * 100 : 0}%`, color: 'bg-green-500', icon: 'box' },
		{ label: 'System Cores', value: `${systemStats.cpus} Cores`, width: '100%', color: 'bg-sky-500', icon: 'wifi' }
	]);

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
	<!-- Top Bar Header -->
	<div class="flex flex-col sm:flex-row sm:items-center justify-between mb-8 gap-4">
		<div>
			<h1 class="text-2xl font-semibold text-gray-900 tracking-tight">Overview</h1>
			<p class="text-gray-500 mt-1">System health and active deployments.</p>
		</div>
		<button
			type="button"
			onclick={onNewService}
			class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors flex items-center justify-center gap-2 shadow-sm"
		>
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
			<span>New Service</span>
		</button>
	</div>

	<!-- System Telemetry Cards -->
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
		{#each stats as stat}
			<div class="bg-white border border-gray-200 rounded-xl p-5 shadow-sm">
				<div class="flex justify-between items-start mb-4">
					<div class="text-sm font-medium text-gray-500">{stat.label}</div>
					<div class="p-2 bg-gray-50 rounded-lg text-gray-400">
						{#if stat.icon === 'cpu'}
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/></svg>
						{:else if stat.icon === 'activity'}
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
						{:else if stat.icon === 'box'}
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
						{:else}
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071a10 10 0 0114.142 0"/></svg>
						{/if}
					</div>
				</div>
				<div class="text-xl font-bold text-gray-900 mb-3">{stat.value}</div>
				<div class="h-2 w-full bg-gray-100 rounded-full overflow-hidden">
					<div class={`h-full ${stat.color} rounded-full transition-all duration-500`} style="width: {stat.width};"></div>
				</div>
			</div>
		{/each}
	</div>

	<!-- Search Input Bar -->
	<div class="flex items-center gap-3 mb-6 bg-white p-2 border border-gray-200 rounded-xl shadow-sm">
		<svg class="w-5 h-5 ml-2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>
		<input
			type="text"
			placeholder="Search services across all active containers..."
			bind:value={search}
			class="w-full bg-transparent border-none focus:ring-0 text-sm text-gray-900 placeholder-gray-500 py-2 outline-none"
		/>
	</div>

	<!-- Services List Group -->
	<div class="space-y-4">
		<div class="flex items-center gap-2 text-sm font-semibold text-gray-700 px-1">
			<svg class="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>
			<span>Services in this workspace</span>
			<span class="bg-gray-100 text-gray-500 px-2 py-0.5 rounded-full text-xs ml-2">
				{containers.filter(c => c.name.toLowerCase().includes(search.toLowerCase())).length}
			</span>
		</div>

		<div class="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden divide-y divide-gray-100">
			{#if containers.length === 0}
				<div class="p-8 text-center text-gray-500">
					This workspace has no services yet. Create a service to deploy an app, database, or background job.
				</div>
			{:else}
				{#each containers.filter(c => c.name.toLowerCase().includes(search.toLowerCase())) as container}
					<div
						role="button"
						tabindex="0"
						class="p-4 sm:p-5 hover:bg-gray-50 transition-colors cursor-pointer group flex flex-col sm:flex-row sm:items-center justify-between gap-4"
						onclick={() => onSelectService(container)}
						onkeydown={(e) => e.key === 'Enter' && onSelectService(container)}
					>
						<div class="flex items-start gap-4 flex-1">
							<div class="p-2.5 bg-gray-50 rounded-lg text-blue-600 border border-gray-200 group-hover:bg-blue-50 transition-colors">
								<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/></svg>
							</div>
							<div>
								<div class="flex items-center gap-3 mb-1">
									<h3 class="font-semibold text-gray-900 group-hover:text-blue-600 transition-colors">
										{container.name}
									</h3>
									<span class="px-2.5 py-0.5 rounded-full text-xs font-medium border uppercase tracking-wider {getStatusBadgeClass(container.status)}">
										{container.status}
									</span>
								</div>
								<div class="flex flex-wrap items-center text-sm text-gray-500 gap-x-4 gap-y-2 font-mono text-xs">
									<span>Image: {container.image}</span>
									<span>Ports: {container.port || 'Auto'}</span>
									<span>CPU: {container.cpuPercent || 0}%</span>
									<span>RAM: {container.memoryMb || 0} MB</span>
								</div>
							</div>
						</div>

						<div class="flex items-center justify-between sm:justify-end gap-6 w-full sm:w-auto pl-12 sm:pl-0">
							<div class="text-sm text-gray-500 hidden md:block text-right">
								<div class="font-medium text-gray-700">Docker Engine</div>
								<div class="font-mono text-xs">{container.uptime || 'Active'}</div>
							</div>

							<div role="none" class="flex items-center gap-2" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
								<DropdownMenu
									right
									trigger={() => (
										`<button type="button" aria-label="Container options" class="p-2 text-gray-400 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors">
											<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"/></svg>
										</button>`
									)}
									items={[
										{ label: container.status === 'running' ? 'Stop' : 'Start', onClick: () => onToggleStatus(container) },
										{ label: 'View Logs', onClick: () => onOpenLogs(container) },
										{ divider: true },
										{ label: 'Delete Container', danger: true, onClick: () => onPromptDelete(container) }
									]}
								/>
							</div>
						</div>
					</div>
				{/each}
			{/if}
		</div>
	</div>
</div>
