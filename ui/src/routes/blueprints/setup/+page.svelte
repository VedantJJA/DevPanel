<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import type { Blueprint } from '$lib/types';

	// Dynamic blueprint state passed from validation or sessionStorage
	let repoUrl = $state('https://github.com/username/my-monorepo.git');
	let blueprint = $state<Blueprint>({
		id: 'bp-monorepo',
		name: 'my-monorepo-startup',
		repo_url: 'https://github.com/username/my-monorepo.git',
		version: '1.0',
		project: 'my-monorepo-startup',
		services: {
			frontend: {
				type: 'static',
				source: {
					directory: 'web',
					ref: 'main'
				},
				build: {
					engine: 'node',
					command: 'npm ci && npm run build'
				},
				deploy: {
					port: 3000
				}
			},
			backend: {
				type: 'web',
				source: {
					directory: 'api',
					ref: 'main'
				},
				build: {
					engine: 'dockerfile',
					dockerfile_path: 'Dockerfile'
				},
				deploy: {
					port: 8080,
					command: './server',
					env: {
						DB_HOST: 'database',
						DB_PORT: '5432',
						PORT: '8080'
					}
				}
			},
			database: {
				type: 'database',
				image: 'postgres:15-alpine',
				deploy: {
					port: 5432,
					env: {
						POSTGRES_DB: 'appdb',
						POSTGRES_USER: 'admin'
					}
				}
			}
		}
	});

	let isDeploying = $state(false);
	let errorMessage = $state<string | null>(null);

	onMount(() => {
		if (typeof window !== 'undefined') {
			const saved = sessionStorage.getItem('devpanel_setup_data');
			if (saved) {
				try {
					const data = JSON.parse(saved);
					if (data.repoUrl) repoUrl = data.repoUrl;
					if (data.blueprint) blueprint = data.blueprint;
					if (data.appName && (!blueprint.project || blueprint.project === 'my-monorepo-startup')) {
						blueprint.project = data.appName;
					}
				} catch (e) {
					console.error('Failed to parse setup data from sessionStorage:', e);
				}
			}
		}
	});

	async function handleConfirmDeploy() {
		isDeploying = true;
		errorMessage = null;

		try {
			const res = await fetch('/api/deployments/trigger', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					project: blueprint.project,
					repo_url: repoUrl,
					blueprint: blueprint
				})
			});

			const data = await res.json();

			if (!res.ok || data.error) {
				throw new Error(data.error || 'Failed to trigger deployment pipeline');
			}

			// Redirect to Live Console page for real-time log streaming
			const targetProjectID = data.project_id || (blueprint.project || 'project').toLowerCase().replace(/[^a-z0-9-]/g, '-');
			goto(`/deployments/${targetProjectID}`);
		} catch (err: any) {
			errorMessage = err.message || 'An unexpected error occurred during deployment submission.';
		} finally {
			isDeploying = false;
		}
	}
</script>

<div class="min-h-screen bg-neutral-950 text-neutral-100 p-6 md:p-12 font-sans antialiased">
	<div class="max-w-5xl mx-auto space-y-8">
		<!-- Navigation & Top Header -->
		<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-neutral-800/80 pb-6">
			<div>
				<div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs font-mono mb-2">
					<span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
					Setup Review Phase
				</div>
				<h1 class="text-3xl font-bold tracking-tight text-neutral-100 sm:text-4xl">Review Application Blueprint</h1>
				<p class="text-xs text-neutral-400 mt-1">Verify parsed architecture and service definitions before provisioning container infrastructure.</p>
			</div>

			<div class="flex items-center gap-3">
				<a href="/blueprints/new" class="px-4 py-2 rounded-xl bg-neutral-900 border border-neutral-800 text-neutral-300 hover:text-neutral-100 text-xs font-semibold transition-all">
					← Edit Repository
				</a>
			</div>
		</div>

		<!-- Error Alert Banner -->
		{#if errorMessage}
			<div class="p-4 rounded-2xl bg-rose-950/60 border border-rose-800/80 text-rose-200 text-sm flex items-center justify-between shadow-xl">
				<div class="flex items-center gap-3">
					<svg class="w-5 h-5 text-rose-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
					<span>{errorMessage}</span>
				</div>
				<button onclick={() => (errorMessage = null)} class="text-rose-400 hover:text-rose-200 text-xs font-bold px-2 py-1">✕</button>
			</div>
		{/if}

		<!-- Blueprint Summary Card -->
		<div class="p-6 rounded-3xl bg-neutral-900/60 border border-neutral-800 shadow-xl backdrop-blur-md flex flex-col md:flex-row md:items-center justify-between gap-6">
			<div class="space-y-1">
				<div class="text-xs uppercase tracking-wider text-neutral-400 font-semibold">Project Name</div>
				<div class="text-2xl font-bold text-neutral-100 font-mono flex items-center gap-3">
					<span>{blueprint.project}</span>
					<span class="text-xs px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-sans font-medium">v{blueprint.version}</span>
				</div>
			</div>

			<div class="space-y-1">
				<div class="text-xs uppercase tracking-wider text-neutral-400 font-semibold">Git Repository Source</div>
				<div class="text-xs font-mono text-emerald-400 truncate max-w-md" title={repoUrl}>
					{repoUrl}
				</div>
			</div>

			<div class="space-y-1">
				<div class="text-xs uppercase tracking-wider text-neutral-400 font-semibold">Configured Services</div>
				<div class="text-lg font-bold text-neutral-100 font-mono">
					{Object.keys(blueprint.services || {}).length} Services
				</div>
			</div>
		</div>

		<!-- Service Cards Grid -->
		<div class="space-y-6">
			<h2 class="text-lg font-bold text-neutral-100 flex items-center gap-2">
				<svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
				Services Architecture
			</h2>

			<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
				{#each Object.entries(blueprint.services || {}) as [svcName, svcCfg] (svcName)}
					<div class="p-6 rounded-3xl bg-neutral-900/60 border border-neutral-800 space-y-5 shadow-xl flex flex-col justify-between hover:border-neutral-700 transition-all">
						<!-- Service Card Header -->
						<div class="flex items-center justify-between border-b border-neutral-800/80 pb-4">
							<div>
								<h3 class="font-bold text-lg text-neutral-100">{svcName}</h3>
								<p class="text-xs text-neutral-400 font-mono">Subdir: {svcCfg.source?.directory || './'}</p>
							</div>

							<span class="px-3 py-1 rounded-full text-xs font-semibold font-mono uppercase tracking-wider border {svcCfg.type === 'static'
								? 'bg-sky-500/10 text-sky-400 border-sky-500/20'
								: svcCfg.type === 'web'
								? 'bg-purple-500/10 text-purple-400 border-purple-500/20'
								: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'}">
								{svcCfg.type}
							</span>
						</div>

						<!-- Attributes Grid -->
						<div class="grid grid-cols-2 gap-4 text-xs font-mono">
							<div>
								<span class="text-neutral-500 block">Repository:</span>
								<span class="text-neutral-300 truncate block">{svcCfg.source?.repo || 'Current Repo'}</span>
							</div>

							<div>
								<span class="text-neutral-500 block">Target Branch:</span>
								<span class="text-neutral-300">{svcCfg.source?.ref || 'main'}</span>
							</div>

							<div>
								<span class="text-neutral-500 block">Build Engine / Cmd:</span>
								<span class="text-emerald-400">{svcCfg.build?.engine || 'dockerfile'} {svcCfg.build?.command ? `(${svcCfg.build.command})` : ''}</span>
							</div>

							<div>
								<span class="text-neutral-500 block">Exposed Port:</span>
								<span class="text-neutral-300">{svcCfg.deploy?.port || 80}</span>
							</div>

							{#if svcCfg.image}
								<div class="col-span-2">
									<span class="text-neutral-500 block">Image Source:</span>
									<span class="text-purple-400 font-bold">{svcCfg.image}</span>
								</div>
							{/if}
						</div>

						<!-- Environment Variables List -->
						<div class="pt-3 border-t border-neutral-800/80 space-y-2">
							<span class="text-xs font-semibold uppercase tracking-wider text-neutral-400 block">Environment Variables</span>
							{#if svcCfg.deploy?.env && Object.keys(svcCfg.deploy.env).length > 0}
								<div class="p-3 rounded-xl bg-neutral-950 border border-neutral-800 text-xs font-mono space-y-1.5 max-h-32 overflow-y-auto">
									{#each Object.entries(svcCfg.deploy.env) as [k, v]}
										<div class="flex items-center justify-between text-neutral-300 border-b border-neutral-800/40 py-0.5 last:border-0">
											<span class="text-neutral-400">{k}</span>
											<span class="text-emerald-400">{v}</span>
										</div>
									{/each}
								</div>
							{:else}
								<span class="text-xs text-neutral-500 italic block">No environment variables configured</span>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</div>

		<!-- Action Footer with Prominent Deploy Button -->
		<div class="pt-6 border-t border-neutral-800/80 flex flex-col sm:flex-row items-center justify-between gap-4">
			<a href="/blueprints/new" class="text-xs font-medium text-neutral-400 hover:text-emerald-400 transition-colors">
				← Back to Blueprint Input
			</a>

			<button
				onclick={handleConfirmDeploy}
				disabled={isDeploying}
				class="w-full sm:w-auto py-4 px-8 rounded-2xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-base flex items-center justify-center gap-3 transition-all shadow-xl shadow-emerald-950/60 disabled:opacity-50"
			>
				{#if isDeploying}
					<svg class="w-5 h-5 animate-spin text-white" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
					<span>Provisioning Container Stack...</span>
				{:else}
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
					<span>Deploy Application</span>
				{/if}
			</button>
		</div>
	</div>
</div>
