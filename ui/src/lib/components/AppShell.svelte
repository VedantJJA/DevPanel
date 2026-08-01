<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { SystemStats } from '$lib/types';
	import { page } from '$app/stores';

	export interface ProjectNavItem {
		id: string;
		label: string;
		icon: string;
	}

	export interface ProjectNavGroup {
		title: string;
		items: ProjectNavItem[];
	}

	export interface ProjectContext {
		projectName: string;
		services?: any[];
		activeServiceIdx?: number;
		onSelectService?: (idx: number) => void;
		activeTab: string;
		onSelectTab: (tabId: string) => void;
		groups: ProjectNavGroup[];
	}

	interface Props {
		children: Snippet;
		systemStats?: SystemStats;
		pingMs?: number | null;
		onLogout?: () => void;
		projectContext?: ProjectContext | null;
	}

	let { children, systemStats, pingMs = null, onLogout, projectContext = null }: Props = $props();

	const nav = [
		{ href: '/', icon: 'dashboard', label: 'Dashboard', exact: true },
		{ href: '/blueprints', icon: 'folder_open', label: 'Projects' },
		{ href: '/containers', icon: 'terminal', label: 'Raw Containers' },
		{ href: '/metrics', icon: 'query_stats', label: 'Metrics' },
		{ href: '/proxy', icon: 'router', label: 'Caddy Proxy' },
		{ href: '/volumes', icon: 'storage', label: 'Storage Volumes' },
		{ href: '/settings', icon: 'settings', label: 'Settings' },
	];

	function isActive(href: string, exact: boolean = false): boolean {
		if (exact) return $page.url.pathname === href;
		return $page.url.pathname.startsWith(href);
	}

	async function handleLogout() {
		await fetch('/api/auth/logout', { method: 'POST' });
		window.location.href = '/login';
	}
</script>

<div class="min-h-screen" style="background-color: var(--background)">
	<!-- Desktop Sidebar -->
	<aside class="fixed left-0 top-0 z-50 hidden h-screen w-[260px] flex-col overflow-y-auto border-r lg:flex" style="background-color: var(--surface-lowest); border-color: var(--outline-variant);">
		<!-- Brand Logo Header -->
		<div class="mb-6 px-6 pt-6">
			<a href="/" class="flex items-center gap-2 text-xl font-bold" style="color: var(--primary)">
				<span class="material-symbols-outlined">dns</span>
				DevPanel
			</a>
			<p class="mt-1 text-xs" style="color: var(--on-surface-variant)">Self-hosted PaaS · v1.0</p>
		</div>

		{#if projectContext}
			<!-- Project-Specific Side Navigation (Stitch Design) -->
			<div class="mb-4 px-4">
				<a href="/blueprints" class="flex items-center gap-2 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors hover:bg-[color:var(--surface-low)]" style="color: var(--on-surface-variant)">
					<span class="material-symbols-outlined" style="font-size: 16px">arrow_back</span>
					Back to System Overview
				</a>
				<div class="mt-3 rounded-lg border p-3" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
					<p class="text-[10px] font-bold uppercase tracking-wider mb-0.5" style="color: var(--on-surface-variant)">Active Project</p>
					<p class="truncate font-bold text-sm mb-2" style="color: var(--primary)">{projectContext.projectName}</p>
					{#if projectContext.services && projectContext.services.length > 0}
						<div class="pt-2 border-t" style="border-color: var(--outline-variant)">
							<label for="sideServiceSelect" class="block text-[10px] font-semibold uppercase tracking-wider mb-1" style="color: var(--on-surface-variant)">Select Service</label>
							<select
								id="sideServiceSelect"
								value={String(projectContext.activeServiceIdx ?? 0)}
								onchange={(e) => {
									const target = e.target as HTMLSelectElement;
									if (target && projectContext?.onSelectService) {
										projectContext.onSelectService(Number(target.value));
									}
								}}
								class="w-full rounded border px-2 py-1.5 text-xs font-semibold shadow-sm outline-none transition-colors cursor-pointer"
								style="border-color: var(--outline-variant); background-color: var(--surface-lowest); color: var(--on-surface)"
							>
								{#each projectContext.services as svc, idx}
									<option value={String(idx)}>{svc.name} ({svc.type})</option>
								{/each}
							</select>
						</div>
					{/if}
				</div>
			</div>

			<nav class="flex-grow space-y-4 px-3">
				{#each projectContext.groups as group}
					<div>
						<p class="mb-1 px-3 text-[10px] font-bold uppercase tracking-[0.1em]" style="color: var(--outline)">{group.title}</p>
						<div class="space-y-0.5">
							{#each group.items as item}
								<button
									onclick={() => projectContext?.onSelectTab(item.id)}
									class="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors text-left {projectContext.activeTab === item.id ? 'font-semibold' : ''}"
									style={projectContext.activeTab === item.id
										? 'background-color: var(--primary-fixed); color: var(--primary); font-weight: 600;'
										: 'color: var(--on-surface-variant);'}
									onmouseenter={(e) => { if (projectContext?.activeTab !== item.id) (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--surface-low)'; }}
									onmouseleave={(e) => { if (projectContext?.activeTab !== item.id) (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}
								>
									<span class="material-symbols-outlined" style="font-size: 18px">{item.icon}</span>
									<span>{item.label}</span>
								</button>
							{/each}
						</div>
					</div>
				{/each}
			</nav>

		{:else}
			<!-- Global System Side Navigation -->
			<nav class="flex-grow">
				<ul class="space-y-1">
					{#each nav as item}
						<li class="px-4">
							<a
								href={item.href}
								class="flex items-center gap-3 rounded-lg px-4 py-2 text-sm transition-colors {isActive(item.href, item.exact)
									? 'font-semibold'
									: ''}"
								style={isActive(item.href, item.exact)
									? 'background-color: var(--primary-fixed); color: var(--primary);'
									: 'color: var(--on-surface-variant);'}
								onmouseenter={(e) => { if (!isActive(item.href, item.exact)) (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--surface-low)'; }}
								onmouseleave={(e) => { if (!isActive(item.href, item.exact)) (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}
							>
								<span class="material-symbols-outlined" style="font-size: 20px">{item.icon}</span>
								{item.label}
							</a>
						</li>
					{/each}
				</ul>
			</nav>
		{/if}

		<!-- Deploy Button + VM Status -->
		<div class="mt-6 px-4 pb-6">
			<a
				href="/new"
				class="flex w-full items-center justify-center gap-2 rounded-lg px-4 py-3 text-sm font-semibold shadow-sm transition-opacity hover:opacity-90"
				style="background-color: var(--primary); color: var(--on-primary);"
			>
				<span class="material-symbols-outlined" style="font-size: 20px">add</span>
				Deploy Service
			</a>
			<div class="mt-4 rounded-lg border p-3" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
				<div class="flex items-center gap-2 text-sm font-semibold" style="color: var(--success)">
					<span class="material-symbols-outlined" style="font-size: 18px">check_circle</span>
					{#if pingMs !== null}
						Backend online · {pingMs}ms
					{:else}
						Checking backend…
					{/if}
				</div>
				<p class="mt-1 text-xs" style="color: var(--on-surface-variant)">
					{systemStats?.os || 'Linux'} · {systemStats?.cpus || 0} vCPU · {Math.round((systemStats?.totalMemMb || 0) / 1024)} GB RAM
				</p>
			</div>
			<!-- Logout -->
			<button
				type="button"
				onclick={handleLogout}
				class="mt-3 flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors hover:opacity-80"
				style="color: var(--on-surface-variant);"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--surface-high)'; (e.currentTarget as HTMLElement).style.color = 'var(--error)'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = ''; (e.currentTarget as HTMLElement).style.color = 'var(--on-surface-variant)'; }}
			>
				<span class="material-symbols-outlined" style="font-size: 18px">logout</span>
				Log out
			</button>
		</div>
	</aside>

	<!-- Top Header Bar -->
	<header class="fixed left-0 right-0 top-0 z-40 flex h-16 items-center justify-between gap-4 border-b px-4 lg:left-[260px] lg:px-6" style="background-color: var(--surface-lowest); border-color: var(--outline-variant);">
		<!-- Mobile Logo -->
		<a href="/" class="text-lg font-bold lg:hidden" style="color: var(--primary)">DevPanel</a>

		<!-- Search -->
		<div class="relative hidden w-full max-w-md sm:block">
			<span class="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2" style="font-size: 20px; color: var(--outline)">search</span>
			<input
				type="text"
				placeholder="Search projects, logs, settings…"
				class="w-full rounded-full border py-2 pl-10 pr-4 text-sm outline-none transition-all"
				style="border-color: var(--outline-variant); background-color: var(--surface-low);"
				onfocus={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--primary)'; (e.currentTarget as HTMLElement).style.boxShadow = '0 0 0 3px color-mix(in oklch, var(--primary), transparent 80%)'; }}
				onblur={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--outline-variant)'; (e.currentTarget as HTMLElement).style.boxShadow = ''; }}
			/>
		</div>

		<!-- Right Actions -->
		<div class="flex items-center gap-4">
			<button class="relative transition-colors hover:opacity-80" style="color: var(--on-surface-variant)">
				<span class="material-symbols-outlined">notifications</span>
				<span class="absolute right-0.5 top-0.5 h-2 w-2 rounded-full" style="background-color: var(--error)"></span>
			</button>
			<div class="hidden h-8 w-px sm:block" style="background-color: var(--outline-variant)"></div>
			<div class="flex items-center gap-2">
				<span class="flex h-8 w-8 items-center justify-center rounded-full text-xs font-bold" style="background-color: var(--primary); color: var(--on-primary)">DP</span>
				<span class="hidden text-sm font-semibold sm:inline" style="color: var(--on-surface)">Production</span>
			</div>
		</div>
	</header>

	<!-- Mobile Bottom Nav -->
	<nav class="fixed bottom-0 left-0 right-0 z-50 flex justify-around border-t py-2 lg:hidden" style="background-color: var(--surface-lowest); border-color: var(--outline-variant);">
		{#if projectContext}
			{#each projectContext.groups.flatMap(g => g.items).slice(0, 5) as item}
				<button
					onclick={() => projectContext?.onSelectTab(item.id)}
					class="flex flex-col items-center gap-0.5 px-2 text-[10px]"
					style={projectContext.activeTab === item.id ? 'color: var(--primary)' : 'color: var(--on-surface-variant)'}
				>
					<span class="material-symbols-outlined" style="font-size: 22px">{item.icon}</span>
					{item.label.split(' ')[0]}
				</button>
			{/each}
		{:else}
			{#each nav.slice(0, 5) as item}
				<a
					href={item.href}
					class="flex flex-col items-center gap-0.5 px-2 text-[10px]"
					style={isActive(item.href, item.exact) ? 'color: var(--primary)' : 'color: var(--on-surface-variant)'}
				>
					<span class="material-symbols-outlined" style="font-size: 22px">{item.icon}</span>
					{item.label.split(' ')[0]}
				</a>
			{/each}
		{/if}
	</nav>

	<!-- Main Content -->
	<main class="min-h-screen pb-20 pt-16 lg:ml-[260px] lg:pb-0">
		<div class="mx-auto max-w-6xl p-4 lg:p-6">
			{@render children()}
		</div>
	</main>
</div>
