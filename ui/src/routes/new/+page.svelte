<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import AppShell from '$lib/components/AppShell.svelte';
	import { scanRepo, createProject, triggerDeploy } from '$lib/api';
	import type { ScanResult, SystemStats } from '$lib/types';

	let step = $state(1); // 1: Select Type, 2: Connect Repo, 3: Configure Service, 4: Review & Deploy
	let selectedType = $state<any>(null);
	let selectedRepo = $state<any>(null);

	// User & Repos State
	let storedGithubUsername = $state('VedantJJA');
	let storedGithubToken = $state('');
	let userRepos = $state<any[]>([]);
	let isFetchingRepos = $state(false);
	let repoFetchError = $state<string | null>(null);
	let isAuthenticated = $state(false);
	let repoSearchQuery = $state('');
	let patInput = $state('');
	let isSavingPat = $state(false);
	let patSaveSuccess = $state<string | null>(null);

	// Service Form Configuration State
	let appName = $state('my-awesome-app');
	let repoUrl = $state('https://github.com/VedantJJA/DevPtestrepo');
	let runtime = $state('Node.js');
	let buildCommand = $state('npm ci && npm run build');
	let startCommand = $state('npm start');
	let publishDir = $state('dist');
	let containerPort = $state(80);
	let dbVersion = $state('15');
	let cronSchedule = $state('0 0 * * *');
	let cronTask = $state('Sync Users');
	let envVars = $state<Record<string, string>>({});

	let isDeploying = $state(false);
	let deployError = $state<string | null>(null);

	// Pre-deployment Review State
	let preDeployData = $state<{
		scanResult: ScanResult | null;
		payloadServices: any[];
		blueprint: any;
	} | null>(null);

	let systemStats = $state<SystemStats>({ totalContainers: 0, activeContainers: 0, stoppedContainers: 0, totalMemMb: 0, usedMemMb: 0, memPercent: 0, cpus: 1 });

	let filteredRepos = $derived(
		userRepos.filter((r) =>
			(r.full_name || r.name || '').toLowerCase().includes(repoSearchQuery.toLowerCase().trim())
		)
	);

	const SERVICE_TYPES = [
		{ id: 'repo', title: 'Deploy from Repository', desc: 'Auto-detect services from devpanel.yaml', icon: 'layers', color: 'text-blue-600', bg: 'bg-blue-50', border: 'border-blue-200', needsRepo: true },
		{ id: 'web', title: 'Web Service', desc: 'Node, Python, Go, Ruby, Docker', icon: 'server', color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: true },
		{ id: 'static', title: 'Static Site', desc: 'React, Vue, Astro, HTML/CSS', icon: 'globe', color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: true },
		{ id: 'postgres', title: 'PostgreSQL', desc: 'Managed relational database', icon: 'database', color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: false },
		{ id: 'redis', title: 'Redis', desc: 'Managed in-memory cache', icon: 'database', color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: false },
		{ id: 'cron', title: 'Cron Job', desc: 'Scheduled tasks and scripts', icon: 'clock', color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: true }
	];

	async function loadGithubSettings() {
		try {
			const res = await fetch('/api/settings');
			if (res.ok) {
				const s = await res.json();
				if (s.github_username) storedGithubUsername = s.github_username;
				if (s.github_token) storedGithubToken = s.github_token;
			}
		} catch (e) {
			console.error('Failed to load settings:', e);
		}
	}

	async function fetchUserRepos(username: string = '') {
		isFetchingRepos = true;
		repoFetchError = null;
		try {
			const u = username.trim() || storedGithubUsername.trim();
			const url = u ? `/api/repos/user?username=${encodeURIComponent(u)}` : '/api/repos/user';
			const res = await fetch(url);
			if (!res.ok) {
				const err = await res.json().catch(() => ({ error: 'Failed to fetch repositories' }));
				throw new Error(err.error || 'Failed to load GitHub repositories');
			}
			const data = await res.json();
			userRepos = data.repos || [];
			isAuthenticated = data.authenticated || false;
		} catch (err: any) {
			repoFetchError = err.message;
		} finally {
			isFetchingRepos = false;
		}
	}

	async function savePatAndAuthorize() {
		if (!patInput.trim()) return;
		isSavingPat = true;
		patSaveSuccess = null;
		try {
			await fetch('/api/settings', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ github_username: storedGithubUsername, github_token: patInput.trim() })
			});
			storedGithubToken = patInput.trim();
			patSaveSuccess = 'GitHub Personal Access Token authorized successfully!';
			patInput = '';
			await fetchUserRepos();
		} catch (e: any) {
			repoFetchError = `Failed to authorize token: ${e.message}`;
		} finally {
			isSavingPat = false;
		}
	}

	function handleTypeSelect(type: any) {
		selectedType = type;
		appName = `my-${type.id}-app`;
		buildCommand = '';
		startCommand = '';

		if (type.id === 'static') {
			containerPort = 80;
			buildCommand = 'npm ci && npm run build';
			publishDir = 'dist';
		} else if (type.id === 'web') {
			containerPort = 8080;
			buildCommand = 'npm ci && npm run build';
			startCommand = 'npm start';
		} else if (type.id === 'postgres') {
			containerPort = 5432;
		} else if (type.id === 'redis') {
			containerPort = 6379;
		}

		if (type.needsRepo) {
			step = 2;
			fetchUserRepos();
		} else {
			step = 3;
		}
	}

	function handleRepoSelect(repo: any) {
		selectedRepo = repo;
		repoUrl = repo.url;
		if (repo.name) {
			appName = repo.name.replace(/[^a-zA-Z0-9-]/g, '-').toLowerCase();
		}
		const lang = (repo.language || repo.name || '').toLowerCase();
		if (lang.includes('python') || lang.includes('py')) {
			runtime = 'Python';
			buildCommand = 'pip install -r requirements.txt';
			startCommand = 'python app.py';
			containerPort = 8080;
		} else if (lang.includes('go')) {
			runtime = 'Go';
			buildCommand = 'go build -o server .';
			startCommand = './server';
			containerPort = 8080;
		} else if (lang.includes('rust')) {
			runtime = 'Rust';
			buildCommand = 'cargo build --release';
			startCommand = './target/release/app';
			containerPort = 8080;
		} else if (lang.includes('ruby')) {
			runtime = 'Ruby';
			buildCommand = 'bundle install';
			startCommand = 'bundle exec rackup -p 8080';
			containerPort = 8080;
		} else {
			runtime = 'Node.js';
			if (selectedType?.id === 'static') {
				buildCommand = 'npm install && npm run build';
				containerPort = 80;
			} else {
				buildCommand = 'npm install && npm run build';
				startCommand = 'npm start';
				containerPort = 8080;
			}
		}
		step = 3;
	}

	async function goToReview() {
		deployError = null;
		if (selectedType?.needsRepo && !repoUrl.trim()) {
			deployError = 'Repository URL is required';
			return;
		}

		isDeploying = true;
		try {
			let scanResult: ScanResult | null = null;

			if (selectedType?.needsRepo && repoUrl.trim()) {
				try {
					scanResult = await scanRepo(repoUrl.trim(), appName.trim());
				} catch (e) {
					console.warn('Repository scan warning:', e);
				}
			}

			let payloadServices: any[] = [];
			if (selectedType?.id === 'repo' && scanResult && scanResult.services && scanResult.services.length > 0) {
				payloadServices = scanResult.services.map((s: any) => ({
					name: s.name || appName.trim() || 'my-service',
					type: s.type || 'web',
					image: s.image || '',
					port: Number(s.default?.port || s.port || (s.type === 'static' ? 80 : 8080)),
					env_vars: s.default?.env || s.env_vars || {},
					build_command: s.buildCommand || s.build?.command || '',
					start_command: s.startCommand || s.deploy?.command || ''
				}));
			} else {
				payloadServices = [
					{
						name: appName.trim() || 'my-app',
						type: selectedType?.id === 'repo' ? 'web' : selectedType?.id,
						port: Number(containerPort) || 80,
						env_vars: envVars || {},
						build_command: buildCommand || '',
						start_command: startCommand || ''
					}
				];
			}

			const blueprint = {
				name: appName.trim() || 'my-blueprint',
				repo_url: selectedType?.needsRepo ? repoUrl.trim() : '',
				services: payloadServices
			};

			preDeployData = { scanResult, payloadServices, blueprint };
			step = 4;
		} catch (err: any) {
			deployError = err.message || 'Failed to prepare deployment preview';
		} finally {
			isDeploying = false;
		}
	}

	async function executeDeploy() {
		if (!preDeployData) return;
		isDeploying = true;
		deployError = null;
		try {
			const project = await createProject({
				app_name: appName.trim() || preDeployData.blueprint.name || 'my-app',
				repo_url: preDeployData.blueprint.repo_url || repoUrl.trim(),
				blueprint: preDeployData.blueprint,
				services: preDeployData.payloadServices
			});
			await triggerDeploy(project.blueprint.id);
			goto(`/projects/${project.blueprint.id}?tab=logs&deploying=true`);
		} catch (err: any) {
			deployError = err.message || 'Deployment execution failed';
			isDeploying = false;
		}
	}

	onMount(async () => {
		await loadGithubSettings();
		try {
			const sRes = await fetch('/api/system/stats');
			if (sRes.ok) {
				const s = await sRes.json();
				systemStats = { ...systemStats, ...s };
			}
		} catch (e) { console.error(e); }
	});
</script>

<AppShell {systemStats}>
<div class="max-w-5xl mx-auto w-full" style="color: var(--on-surface)">
	<button
		type="button"
		onclick={() => {
			if (step > 1) {
				if (step === 3 && selectedType?.needsRepo) step = 2;
				else step = 1;
			} else {
				goto('/');
			}
		}}
		class="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-900 transition-colors mb-8 font-medium"
	>
		<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/></svg>
		<span>{step === 1 ? 'Back to Dashboard' : 'Back'}</span>
	</button>

	<!-- Step 1: Select Type -->
	{#if step === 1}
		<div class="animate-in fade-in slide-in-from-bottom-4 duration-500">
			<h1 class="text-3xl font-bold text-gray-900 mb-2 tracking-tight">Deploy a New Resource</h1>
			<p class="text-gray-500 mb-10 text-lg">Select a service type or blueprint to deploy to your infrastructure.</p>

			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
				{#each SERVICE_TYPES as item}
					<div
						class={`bg-white border ${item.border} hover:border-blue-300 hover:shadow-md rounded-xl p-6 cursor-pointer transition-all hover:-translate-y-1 group flex flex-col h-full`}
						onclick={() => handleTypeSelect(item)}
						role="button"
						tabindex="0"
						onkeydown={(e) => e.key === 'Enter' && handleTypeSelect(item)}
					>
						<div class={`w-12 h-12 rounded-lg ${item.bg} ${item.color} flex items-center justify-center mb-6 transition-colors group-hover:bg-blue-600 group-hover:text-white`}>
							{#if item.icon === 'layers'}
								<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
							{:else if item.icon === 'server'}
								<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/></svg>
							{:else if item.icon === 'globe'}
								<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0zM3.6 9h16.8M3.6 15h16.8"/></svg>
							{:else if item.icon === 'database'}
								<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8-4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"/></svg>
							{:else}
								<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
							{/if}
						</div>
						<h3 class="text-lg font-semibold text-gray-900 mb-2 group-hover:text-blue-600 transition-colors">{item.title}</h3>
						<p class="text-gray-500 text-sm flex-1">{item.desc}</p>
						<div class="mt-6 flex items-center gap-2 text-sm font-medium text-gray-400 group-hover:text-blue-600 transition-colors">
							<span>Continue</span>
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Step 2: Select Source Repository -->
	{#if step === 2}
		<div class="max-w-3xl mx-auto animate-in fade-in slide-in-from-bottom-4 duration-500 space-y-6">
			<div>
				<h2 class="text-2xl font-bold text-gray-900">Connect a repository</h2>
				<p class="text-sm text-gray-500 mt-1">
					Select a repository from your connected GitHub account or authorize access via Personal Access Token.
				</p>
			</div>

			<!-- GitHub Account Status & PAT Authorization Bar -->
			<div class="bg-white border border-gray-200 rounded-xl overflow-hidden shadow-sm p-4 space-y-4">
				<div class="flex flex-wrap items-center justify-between gap-3">
					<div class="flex items-center gap-2 text-sm font-medium text-gray-700">
						<svg class="w-5 h-5 text-gray-800" fill="currentColor" viewBox="0 0 24 24"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"/></svg>
						<span>GitHub Account</span>
						{#if isAuthenticated}
							<span class="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700 ring-1 ring-inset ring-green-600/20">Authenticated (Private Repos &amp; Webhooks Enabled)</span>
						{:else}
							<span class="inline-flex items-center rounded-md bg-yellow-50 px-2 py-1 text-xs font-medium text-yellow-800 ring-1 ring-inset ring-yellow-600/20">Public Repos Only</span>
						{/if}
					</div>
					{#if storedGithubUsername}
						<span class="text-sm font-mono text-gray-500">@{storedGithubUsername}</span>
					{/if}
				</div>

				{#if patSaveSuccess}
					<div class="text-xs text-green-700 bg-green-50 p-2.5 rounded-lg border border-green-200">
						{patSaveSuccess}
					</div>
				{/if}

				{#if !isAuthenticated}
					<div class="pt-3 border-t border-gray-100 space-y-2">
						<label for="ghPatQuick" class="block text-xs font-semibold text-gray-700">Authorize App &amp; Enable Webhooks (GitHub Personal Access Token)</label>
						<div class="flex gap-2">
							<input
								id="ghPatQuick"
								type="password"
								bind:value={patInput}
								placeholder="ghp_… (Personal Access Token with repo scope)"
								class="flex-1 rounded-lg border border-gray-300 p-2 text-xs font-mono outline-none focus:border-blue-500"
							/>
							<button
								type="button"
								onclick={savePatAndAuthorize}
								disabled={isSavingPat || !patInput.trim()}
								class="rounded-lg bg-blue-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-blue-700 disabled:opacity-50"
							>
								{isSavingPat ? 'Authorizing...' : 'Authorize & Fetch Repos'}
							</button>
						</div>
					</div>
				{/if}
			</div>

			<!-- Search Filter & Repo List Card -->
			<div class="bg-white border border-gray-200 rounded-xl overflow-hidden shadow-sm">
				<div class="p-4 border-b border-gray-200 bg-gray-50 flex items-center justify-between gap-4">
					<div class="relative flex-1">
						<svg class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>
						<input
							type="text"
							bind:value={repoSearchQuery}
							placeholder="Search repositories by name..."
							class="w-full pl-9 pr-4 py-2 bg-white border border-gray-300 rounded-lg text-sm outline-none focus:border-blue-500"
						/>
					</div>
					<button
						type="button"
						onclick={() => fetchUserRepos()}
						class="text-xs text-blue-600 font-semibold hover:underline flex items-center gap-1"
					>
						<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
						Refresh
					</button>
				</div>

				{#if isFetchingRepos}
					<div class="p-8 text-center text-gray-500">
						<svg class="w-6 h-6 animate-spin mx-auto text-blue-600 mb-2" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
						<span>Fetching repositories from GitHub...</span>
					</div>
				{:else if repoFetchError}
					<div class="p-6 text-center text-red-600 bg-red-50 text-sm">
						{repoFetchError}
					</div>
				{:else if filteredRepos.length === 0}
					<div class="p-8 text-center text-gray-500 text-sm">
						{userRepos.length === 0 ? 'No repositories found for this account. Enter a direct URL below or authorize token above.' : `No repositories match "${repoSearchQuery}".`}
					</div>
				{:else}
					<div class="divide-y divide-gray-100 max-h-80 overflow-y-auto">
						{#each filteredRepos as repo}
							<div class="p-4 flex items-center justify-between hover:bg-gray-50 transition-colors">
								<div class="flex items-center gap-3">
									<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/></svg>
									<div>
										<div class="flex items-center gap-2">
											<span class="font-medium text-gray-900">{repo.full_name || repo.name}</span>
											{#if repo.private}
												<span class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-semibold text-gray-600">Private</span>
											{/if}
										</div>
										<div class="text-xs text-gray-500">{repo.description || `Updated ${repo.updated}`}</div>
									</div>
								</div>
								<button
									type="button"
									onclick={() => handleRepoSelect(repo)}
									class="px-4 py-1.5 bg-blue-600 text-white text-xs font-semibold rounded-lg hover:bg-blue-700 transition-colors shadow-sm"
								>
									Connect Repository
								</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Direct Repository URL Fallback -->
			<div class="bg-gray-50 border border-gray-200 rounded-xl p-5 space-y-3">
				<h3 class="text-sm font-semibold text-gray-900">Or enter a public repository URL directly</h3>
				<div class="flex gap-2">
					<input
						type="text"
						bind:value={repoUrl}
						placeholder="https://github.com/username/repository"
						class="flex-1 rounded-lg border border-gray-300 p-2.5 text-sm font-mono outline-none focus:border-blue-500"
					/>
					<button
						type="button"
						onclick={() => (step = 3)}
						class="px-5 py-2.5 bg-gray-900 text-white text-sm font-semibold rounded-lg hover:bg-gray-800 transition-colors"
					>
						Use Direct URL
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Step 3: Configure Service -->
	{#if step === 3}
		<div class="max-w-2xl mx-auto bg-white border border-gray-200 rounded-xl p-8 shadow-sm animate-in fade-in slide-in-from-bottom-4 duration-500 space-y-6">
			<div>
				<h2 class="text-2xl font-bold text-gray-900">Configure Service</h2>
				<p class="text-sm text-gray-500 mt-1">Set up execution scripts, build settings, and target ports.</p>
			</div>

			{#if deployError}
				<div class="p-4 rounded-lg bg-red-50 border border-red-200 text-sm text-red-700">
					{deployError}
				</div>
			{/if}

			<div class="space-y-4">
				<div>
					<label for="appNameInput" class="block text-sm font-medium text-gray-700 mb-1">Application / Service Name</label>
					<input id="appNameInput" type="text" bind:value={appName} class="w-full rounded-lg border border-gray-300 p-2.5 text-sm font-mono outline-none focus:border-blue-500" />
				</div>

				{#if selectedType?.needsRepo}
					<div>
						<label for="repoUrlInput" class="block text-sm font-medium text-gray-700 mb-1">Repository URL</label>
						<input id="repoUrlInput" type="text" bind:value={repoUrl} class="w-full rounded-lg border border-gray-300 p-2.5 text-sm font-mono outline-none focus:border-blue-500" />
					</div>

					{#if selectedType?.id === 'web' || selectedType?.id === 'static'}
						<div class="border-t border-b border-gray-100 py-4 space-y-3">
							<label id="newRuntimeLabel" class="block text-xs font-bold text-gray-700 uppercase tracking-wider">Select Runtime Engine &amp; Version</label>
							<div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
								{#each [
									{ id: 'Node.js', label: 'Node.js', defaultBuild: 'npm install && npm run build', defaultStart: 'npm start', desc: 'Node 20-alpine / 22' },
									{ id: 'Python', label: 'Python 3', defaultBuild: 'pip install -r requirements.txt', defaultStart: 'python app.py', desc: 'Python 3.11-slim' },
									{ id: 'Go', label: 'Go', defaultBuild: 'go build -o server .', defaultStart: './server', desc: 'Go 1.22-alpine' },
									{ id: 'Rust', label: 'Rust', defaultBuild: 'cargo build --release', defaultStart: './target/release/app', desc: 'Rust 1.77-slim' },
									{ id: 'Ruby', label: 'Ruby', defaultBuild: 'bundle install', defaultStart: 'bundle exec rackup -p 8080', desc: 'Ruby 3.3-slim' },
									{ id: 'Docker', label: 'Custom Dockerfile', defaultBuild: '', defaultStart: '', desc: 'Custom Container' }
								] as rt}
									<button
										type="button"
										onclick={() => {
											runtime = rt.id;
											if (rt.defaultBuild) buildCommand = rt.defaultBuild;
											if (rt.defaultStart) startCommand = rt.defaultStart;
											if (selectedType?.id === 'static') containerPort = 80;
										}}
										class="p-3 border rounded-lg text-left transition-all flex flex-col gap-1"
										style={runtime === rt.id
											? 'border-color: #2563eb; background-color: #eff6ff;'
											: 'border-color: #e5e7eb; background-color: #ffffff;'}
									>
										<span class="font-bold text-xs text-gray-900">{rt.label}</span>
										<span class="text-[10px] text-gray-500">{rt.desc}</span>
									</button>
								{/each}
							</div>
						</div>
					{/if}

					{#if selectedType?.id !== 'repo'}
						<div>
							<label for="buildCmdInput" class="block text-sm font-medium text-gray-700 mb-1">Build Command</label>
							<input id="buildCmdInput" type="text" bind:value={buildCommand} placeholder="npm ci && npm run build" class="w-full rounded-lg border border-gray-300 p-2.5 text-sm font-mono outline-none focus:border-blue-500" />
						</div>

						{#if selectedType?.id !== 'static'}
							<div>
								<label for="startCmdInput" class="block text-sm font-medium text-gray-700 mb-1">Start Command</label>
								<input id="startCmdInput" type="text" bind:value={startCommand} placeholder="npm start" class="w-full rounded-lg border border-gray-300 p-2.5 text-sm font-mono outline-none focus:border-blue-500" />
							</div>
						{/if}
					{/if}
				{/if}

				{#if selectedType?.id !== 'repo'}
					<div>
						<label for="portInput" class="block text-sm font-medium text-gray-700 mb-1">Container Port</label>
						<input id="portInput" type="number" bind:value={containerPort} class="w-full rounded-lg border border-gray-300 p-2.5 text-sm font-mono outline-none focus:border-blue-500" />
					</div>
				{:else}
					<div class="rounded-lg bg-blue-50 border border-blue-200 p-4 text-xs text-blue-800 space-y-1">
						<div class="font-bold flex items-center gap-1.5">
							<svg class="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
							<span>Blueprint Auto-Configuration</span>
						</div>
						<p>Build commands, start commands, container ports, and service dependencies will be automatically detected and configured from <code class="font-mono bg-blue-100 px-1 rounded text-blue-900">devpanel.yaml</code> in the repository root.</p>
					</div>
				{/if}
			</div>

			<div class="flex justify-end pt-4 border-t border-gray-100">
				<button
					type="button"
					onclick={goToReview}
					disabled={isDeploying}
					class="px-6 py-2.5 bg-blue-600 text-white font-semibold text-sm rounded-lg hover:bg-blue-700 transition-colors shadow-sm disabled:opacity-50"
				>
					{isDeploying ? 'Preparing Deployment...' : 'Continue to Review'}
				</button>
			</div>
		</div>
	{/if}

	<!-- Step 4: Pre-deployment Review & Confirm -->
	{#if step === 4 && preDeployData}
		<div class="max-w-3xl mx-auto bg-white border border-gray-200 rounded-xl p-8 shadow-sm animate-in fade-in slide-in-from-bottom-4 duration-500 space-y-6">
			<div>
				<h2 class="text-2xl font-bold text-gray-900">Review &amp; Deploy</h2>
				<p class="text-sm text-gray-500 mt-1">Confirm detected services and container settings before triggering build.</p>
			</div>

			{#if deployError}
				<div class="p-4 rounded-lg bg-red-50 border border-red-200 text-sm text-red-700">
					{deployError}
				</div>
			{/if}

			<div class="space-y-4">
				<div class="rounded-lg bg-gray-50 p-4 border border-gray-200">
					<div class="text-xs text-gray-500 font-semibold uppercase tracking-wider mb-2">Blueprint Summary</div>
					<div class="text-sm font-bold text-gray-900">{preDeployData.blueprint.name}</div>
					{#if preDeployData.blueprint.repo_url}
						<div class="text-xs font-mono text-blue-600 mt-1">{preDeployData.blueprint.repo_url}</div>
					{/if}
				</div>

				<div>
					<div class="text-xs text-gray-500 font-semibold uppercase tracking-wider mb-2">Services to be Created ({preDeployData.payloadServices.length})</div>
					<div class="space-y-2">
						{#each preDeployData.payloadServices as svc}
							<div class="p-3 border border-gray-200 rounded-lg flex items-center justify-between font-mono text-xs bg-white">
								<div>
									<span class="font-bold text-gray-900">{svc.name}</span>
									<span class="ml-2 rounded px-1.5 py-0.5 bg-gray-100 text-gray-700 uppercase font-semibold text-[10px]">{svc.type}</span>
								</div>
								<div class="text-gray-500">Port {svc.port}</div>
							</div>
						{/each}
					</div>
				</div>
			</div>

			<div class="flex justify-end gap-3 pt-4 border-t border-gray-100">
				<button
					type="button"
					onclick={() => (step = 3)}
					class="px-5 py-2.5 text-gray-700 font-semibold text-sm rounded-lg border border-gray-300 hover:bg-gray-50 transition-colors"
				>
					Back to Edit
				</button>
				<button
					type="button"
					onclick={executeDeploy}
					disabled={isDeploying}
					class="flex items-center gap-2 px-6 py-2.5 bg-blue-600 text-white font-semibold text-sm rounded-lg hover:bg-blue-700 transition-colors shadow-sm disabled:opacity-50"
				>
					<span class="material-symbols-outlined" style="font-size: 18px">rocket_launch</span>
					{isDeploying ? 'Deploying...' : 'Confirm & Deploy'}
				</button>
			</div>
		</div>
	{/if}
</div>
</AppShell>
