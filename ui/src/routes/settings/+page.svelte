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

	// Derived helpers for live URL preview
	function isLocalDomain(d: string) {
		const h = d.split(':')[0];
		return h === 'localhost' || h === '127.0.0.1' || /^192\.168\./.test(h) || /^10\./.test(h);
	}
	let previewScheme = $derived(isLocalDomain(baseDomain) ? 'http' : 'https');
	let previewProjectUrl = $derived(`${previewScheme}://my-project.${baseDomain}/`);


	async function fetchSettings() {
		try {
			const res = await fetch('/api/settings');
			if (res.ok) {
				const s = await res.json();
				if (s.github_username) githubUsername = s.github_username;
				if (s.github_token) githubToken = s.github_token;
				if (s.routing_mode) routingMode = s.routing_mode;
				baseDomain = s.base_domain || (typeof window !== 'undefined' ? window.location.host.replace(/^panel\./, '') : 'localhost:8090');
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
			body: JSON.stringify({ routing_mode: routingMode })
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
						<h2 class="font-bold text-base">Project Hosting & Domain Routing</h2>
						<p class="text-sm" style="color: var(--on-surface-variant)">Select how hosted projects are accessed publicly. Server domain and IP are auto-detected.</p>
					</div>

					<!-- Step 1: Auto-Detected Panel Domain -->
					<div class="space-y-2">
						<div class="block text-sm font-semibold">
							<span class="flex items-center gap-2">
								<span class="flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold text-white" style="background-color: var(--primary)">1</span>
								Server Domain / Public Address (Auto-Detected)
							</span>
						</div>
						<div class="flex items-center justify-between rounded-lg border p-3 text-sm font-mono" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<div class="flex items-center gap-2">
								<span class="material-symbols-outlined text-primary" style="font-size: 18px">language</span>
								<span class="font-bold" style="color: var(--on-surface)">{baseDomain}</span>
							</div>
							<span class="text-xs font-semibold px-2.5 py-0.5 rounded-full" style="background-color: var(--primary-fixed); color: var(--primary)">Auto-Detected</span>
						</div>
						<p class="text-xs" style="color: var(--on-surface-variant)">
							DevPanel automatically retrieves your domain or public IP from browser connections (<code>window.location.host</code> / <code>Host</code> header).
						</p>

						<!-- Live URL preview -->
						<div class="mt-2 flex items-center gap-2 rounded-lg border p-3 text-xs" style="border-color: var(--outline-variant); background-color: var(--surface-low)">
							<span class="material-symbols-outlined shrink-0" style="font-size: 16px; color: var(--primary)">link</span>
							<span style="color: var(--on-surface-variant)">Generated project URL:</span>
							<code class="font-mono font-bold truncate" style="color: var(--primary)">{previewProjectUrl}</code>
						</div>
					</div>

					<!-- Coolify-style Universal Domain Routing Info -->
					<div class="space-y-2">
						<label id="routing-label" class="block text-sm font-semibold">
							<span class="flex items-center gap-2">
								<span class="flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold text-white" style="background-color: var(--primary)">2</span>
								Coolify Domain Hosting & Proxy Architecture
							</span>
						</label>
						<div class="p-4 border border-emerald-500/30 rounded-xl text-left flex flex-col gap-2 bg-emerald-500/5">
							<div class="flex items-center justify-between font-bold text-sm text-emerald-400">
								<span>Domain-First Host Proxying</span>
								<span class="text-[10px] bg-emerald-500/20 text-emerald-300 px-2 py-0.5 rounded font-bold uppercase border border-emerald-500/30">Active</span>
							</div>
							<code class="text-xs font-mono font-bold text-emerald-300">http://&lt;service-slug&gt;.{baseDomain}</code>
							<p class="text-xs" style="color: var(--on-surface-variant)">
								Every service is assigned an FQDN. Incoming requests route directly by Host header to container ports. Single-domain multi-service projects automatically route <code>/api/*</code> requests to backend containers.
							</p>
						</div>
					</div>

					<button onclick={saveRoutingSettings} class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-opacity hover:opacity-90" style="background-color: var(--primary); color: var(--on-primary)">
						<span class="material-symbols-outlined" style="font-size: 18px">save</span>Save Server Domain
					</button>
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
