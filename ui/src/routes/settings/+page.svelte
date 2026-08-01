<script lang="ts">
	import { onMount } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import type { SystemStats } from '$lib/types';

	let systemStats = $state<SystemStats>({ totalContainers: 0, activeContainers: 0, stoppedContainers: 0, totalMemMb: 0, usedMemMb: 0, memPercent: 0, cpus: 1 });

	const sections = ['Host VM', 'Project Routing', 'GitHub', 'Access tokens', 'Notifications', 'Maintenance'] as const;
	type Section = (typeof sections)[number];
	let active = $state<Section>('Host VM');

	let githubUsername = $state('');
	let githubToken = $state('');
	let routingMode = $state<'path' | 'subdomain'>('path');
	let baseDomain = $state('localhost:8090');

	let autoRefreshSec = $state(5);
	let actionLoading = $state<string | null>(null);
	let saveMessage = $state<string | null>(null);

	async function fetchSettings() {
		try {
			const res = await fetch('/api/settings');
			if (res.ok) {
				const s = await res.json();
				if (s.github_username) githubUsername = s.github_username;
				if (s.github_token) githubToken = s.github_token;
				if (s.routing_mode) routingMode = s.routing_mode;
				if (s.base_domain) baseDomain = s.base_domain;
			}
			const sRes = await fetch('/api/system/stats');
			if (sRes.ok) { const s = await sRes.json(); systemStats = { ...systemStats, ...s }; }
		} catch (e) { console.error(e); }
	}

	async function saveGithub() {
		await fetch('/api/settings', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ github_username: githubUsername, github_token: githubToken })
		});
		if (typeof window !== 'undefined') localStorage.setItem('devpnl_gh_username', githubUsername);
		saveMessage = 'GitHub settings saved!';
		setTimeout(() => (saveMessage = null), 3000);
	}

	async function saveRoutingSettings() {
		await fetch('/api/settings', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ routing_mode: routingMode, base_domain: baseDomain })
		});
		saveMessage = 'Project routing settings saved successfully!';
		setTimeout(() => (saveMessage = null), 3000);
	}

	async function handlePrune() {
		if (!confirm('Prune stopped containers, dangling images and build cache? This cannot be undone.')) return;
		actionLoading = 'prune';
		try {
			await fetch('/api/system/prune', { method: 'POST' });
		} catch (e) { console.error(e); } finally { actionLoading = null; }
	}

	onMount(fetchSettings);

	const inputStyle = `width: 100%; border: 1px solid var(--outline-variant); background-color: var(--surface-lowest); border-radius: 0.5rem; padding: 0.5rem 0.75rem; font-size: 0.875rem; outline: none; transition: border-color 0.15s;`;
</script>

<AppShell {systemStats}>
	<!-- Header -->
	<div class="mb-6">
		<h1 class="text-[28px] font-bold leading-tight lg:text-[32px]" style="color: var(--on-surface)">Settings</h1>
		<p class="mt-1" style="color: var(--on-surface-variant)">Configure the machine DevPanel manages, hosting routes, and API access.</p>
	</div>

	{#if saveMessage}
		<div class="mb-6 flex items-center justify-between rounded-xl border px-4 py-3 text-sm" style="background-color: var(--success-container); border-color: var(--success); color: var(--on-success-container)">
			<span>{saveMessage}</span>
			<button onclick={() => (saveMessage = null)} style="color: var(--on-success-container)">✕</button>
		</div>
	{/if}

	<div class="grid grid-cols-1 gap-6 lg:grid-cols-4">
		<!-- Sidebar Nav -->
		<nav class="card-surface h-fit p-2 lg:col-span-1">
			{#each sections as s}
				<button
					onclick={() => (active = s)}
					class="block w-full rounded-lg px-3 py-2 text-left text-sm transition-colors"
					style={active === s
						? 'background-color: var(--primary-fixed); color: var(--primary); font-weight: 600;'
						: 'color: var(--on-surface-variant);'}
					onmouseenter={(e) => { if (active !== s) (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--surface-low)'; }}
					onmouseleave={(e) => { if (active !== s) (e.currentTarget as HTMLElement).style.backgroundColor = ''; }}
				>
					{s}
				</button>
			{/each}
		</nav>

		<!-- Content Panels -->
		<div class="flex flex-col gap-6 lg:col-span-3">

			{#if active === 'Host VM'}
				<section class="card-surface p-5">
					<h2 class="font-bold">Connected machine</h2>
					<p class="mb-4 text-sm" style="color: var(--on-surface-variant)">DevPanel agent runs on this machine and executes every action.</p>
					<dl class="space-y-3 text-sm">
						{#each [
							['OS', systemStats.os || 'Linux'],
							['Architecture', systemStats.arch || '—'],
							['CPU cores', String(systemStats.cpus)],
							['Total RAM', `${(systemStats.totalMemMb / 1024).toFixed(1)} GB`],
							['Containers', `${systemStats.activeContainers} running / ${systemStats.totalContainers} total`],
						] as [k, v]}
							<div class="flex justify-between gap-4 border-b pb-2 last:border-0" style="border-color: var(--outline-variant)">
								<dt style="color: var(--on-surface-variant)">{k}</dt>
								<dd class="text-right font-medium">{v}</dd>
							</div>
						{/each}
					</dl>
				</section>
			{/if}

			{#if active === 'Project Routing'}
				<section class="card-surface p-5 space-y-6">
					<div>
						<h2 class="font-bold text-base">Project Hosting &amp; Domain Routing</h2>
						<p class="text-sm" style="color: var(--on-surface-variant)">Select how hosted services and projects are addressed across your local machine and Oracle Cloud Server.</p>
					</div>

					<div class="space-y-4">
						<div>
							<label id="routing-label" class="block text-sm font-semibold mb-2">Routing URL Pattern</label>
							<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
								<button
									type="button"
									onclick={() => (routingMode = 'path')}
									class="p-4 border-2 rounded-xl text-left transition-all flex flex-col gap-1"
									style={routingMode === 'path' ? 'border-color: var(--primary); background-color: var(--surface-low);' : 'border-color: var(--outline-variant); background-color: var(--surface-lowest);'}
								>
									<div class="flex items-center justify-between font-bold text-sm">
										<span>Path-Based Routing</span>
										{#if routingMode === 'path'}<span class="text-[10px] bg-primary text-white px-2 py-0.5 rounded font-bold uppercase">Active</span>{/if}
									</div>
									<code class="text-xs font-mono font-bold mt-1 text-blue-600">&lt;domain&gt;/app/&lt;project&gt;</code>
									<p class="text-xs mt-1" style="color: var(--on-surface-variant)">Works instantly without DNS or wildcard SSL setup. Ideal for IP addresses or single domains (e.g. http://localhost:8090/app/my-project).</p>
								</button>

								<button
									type="button"
									onclick={() => (routingMode = 'subdomain')}
									class="p-4 border-2 rounded-xl text-left transition-all flex flex-col gap-1"
									style={routingMode === 'subdomain' ? 'border-color: var(--primary); background-color: var(--surface-low);' : 'border-color: var(--outline-variant); background-color: var(--surface-lowest);'}
								>
									<div class="flex items-center justify-between font-bold text-sm">
										<span>Subdomain Routing</span>
										{#if routingMode === 'subdomain'}<span class="text-[10px] bg-primary text-white px-2 py-0.5 rounded font-bold uppercase">Active</span>{/if}
									</div>
									<code class="text-xs font-mono font-bold mt-1 text-blue-600">&lt;project&gt;.&lt;domain&gt;</code>
									<p class="text-xs mt-1" style="color: var(--on-surface-variant)">Isolated subdomains per service/project. Requires wildcard DNS or hosts entry (e.g. http://my-project.localhost:8090).</p>
								</button>
							</div>
						</div>

						<div>
							<label for="baseDomainInput" class="mb-1.5 block text-sm font-medium">Server Base Domain / Host Address</label>
							<input id="baseDomainInput" type="text" bind:value={baseDomain} placeholder="e.g. localhost:8090 or 129.146.xxx.xxx or my-domain.com" style={inputStyle}
								onfocus={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--primary)'; }}
								onblur={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--outline-variant)'; }} />
							<p class="text-xs mt-1" style="color: var(--on-surface-variant)">When deployed on your Oracle Cloud server, set this to your server's public IP address or domain name.</p>
						</div>

						<button onclick={saveRoutingSettings} class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-opacity hover:opacity-90" style="background-color: var(--primary); color: var(--on-primary)">
							<span class="material-symbols-outlined" style="font-size: 18px">save</span>Save Routing Settings
						</button>
					</div>
				</section>
			{/if}

			{#if active === 'GitHub'}
				<section class="card-surface p-5">
					<h2 class="font-bold">GitHub Account Integration</h2>
					<p class="mb-4 text-sm" style="color: var(--on-surface-variant)">Connect your GitHub account to browse and deploy from your repositories.</p>
					<div class="space-y-4">
						<div>
							<label for="ghUser" class="mb-1.5 block text-sm font-medium">GitHub Username</label>
							<input id="ghUser" type="text" bind:value={githubUsername} placeholder="e.g. VedantJJA" style={inputStyle}
								onfocus={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--primary)'; }}
								onblur={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--outline-variant)'; }} />
						</div>
						<div>
							<label for="ghToken" class="mb-1.5 block text-sm font-medium">Personal Access Token</label>
							<input id="ghToken" type="password" bind:value={githubToken} placeholder="ghp_…" style={inputStyle}
								onfocus={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--primary)'; }}
								onblur={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--outline-variant)'; }} />
						</div>
						<button onclick={saveGithub} class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-opacity hover:opacity-90" style="background-color: var(--primary); color: var(--on-primary)">
							<span class="material-symbols-outlined" style="font-size: 18px">save</span>Save GitHub settings
						</button>
					</div>
				</section>
			{/if}

			{#if active === 'Access tokens'}
				<section class="card-surface p-5">
					<h2 class="font-bold">API Tokens</h2>
					<p class="mb-4 text-sm" style="color: var(--on-surface-variant)">Use these with the CLI or CI to trigger deploys without SSH.</p>
					<div class="py-8 text-center text-sm" style="color: var(--on-surface-variant)">
						<span class="material-symbols-outlined mb-2 block" style="font-size: 36px; color: var(--outline)">key</span>
						Token management coming soon.
					</div>
					<button class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-opacity hover:opacity-90" style="background-color: var(--primary); color: var(--on-primary)">
						<span class="material-symbols-outlined" style="font-size: 18px">add</span>Generate token
					</button>
				</section>
			{/if}

			{#if active === 'Notifications'}
				<section class="card-surface p-5">
					<h2 class="font-bold">Alerts</h2>
					<p class="mb-4 text-sm" style="color: var(--on-surface-variant)">Choose what DevPanel should notify you about.</p>
					{#each [
						['Deploy failed', true],
						['Container restart loop', true],
						['Volume above 80% capacity', true],
						['TLS certificate renewal', false],
						['Weekly usage summary', false],
					] as [label, on]}
						<label class="flex items-center justify-between border-b py-3 last:border-0" style="border-color: var(--outline-variant)">
							<span class="text-sm">{label}</span>
							<input type="checkbox" checked={Boolean(on)} class="h-4 w-4" style="accent-color: var(--primary)" />
						</label>
					{/each}
				</section>
			{/if}

			{#if active === 'Maintenance'}
				<section class="card-surface p-5">
					<h2 class="font-bold">Docker System Maintenance</h2>
					<p class="mb-4 text-sm" style="color: var(--on-surface-variant)">Housekeeping tasks that would normally need a terminal.</p>
					<div class="flex flex-wrap gap-3">
						<button onclick={handlePrune} disabled={actionLoading === 'prune'}
							class="flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors disabled:opacity-50"
							style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
							<span class="material-symbols-outlined" style="font-size: 18px">cleaning_services</span>
							{actionLoading === 'prune' ? 'Pruning…' : 'Prune unused images'}
						</button>
					</div>
				</section>
			{/if}

		</div>
	</div>
</AppShell>
