<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import EnvVarEditor from '$lib/components/EnvVarEditor.svelte';
	import { scanRepo, createProject, triggerDeploy } from '$lib/api';
	import type { ScanResult } from '$lib/types';

	let step = $state<1 | 2 | 3>(1);

	// Service types configuration matching template
	const SERVICE_TYPES = [
		{ id: 'blueprint', title: 'Deploy Blueprint', desc: 'Use pre-configured infra templates', icon: 'layers', color: 'text-blue-600', bg: 'bg-blue-50', border: 'border-blue-200', needsRepo: false },
		{ id: 'web', title: 'Web Service', desc: 'Node, Python, Go, Ruby, Docker', icon: 'server', color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: true },
		{ id: 'static', title: 'Static Site', desc: 'React, Vue, Astro, HTML/CSS', icon: 'globe', color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: true },
		{ id: 'postgres', title: 'PostgreSQL', desc: 'Managed relational database', icon: 'database', color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: false },
		{ id: 'redis', title: 'Redis', desc: 'Managed in-memory cache', icon: 'database', color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: false },
		{ id: 'cron', title: 'Cron Job', desc: 'Scheduled tasks and scripts', icon: 'clock', color: 'text-gray-700', bg: 'bg-gray-100', border: 'border-gray-200', needsRepo: true }
	];

	// Selection State
	let selectedType = $state<any>(null);
	let selectedRepo = $state<any>(null);

	// User & Repos State
	let storedGithubUsername = $state('');
	let userRepos = $state<any[]>([]);
	let isFetchingRepos = $state(false);
	let repoFetchError = $state<string | null>(null);
	let isAuthenticated = $state(false);

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
	let cronCommand = $state('python sync.py');
	let envVars = $state<Record<string, string>>({});

	let isDeploying = $state(false);
	let deployError = $state<string | null>(null);

	async function fetchUserRepos(username: string) {
		if (!username.trim()) return;
		isFetchingRepos = true;
		repoFetchError = null;
		try {
			const res = await fetch(`/api/repos/user?username=${encodeURIComponent(username.trim())}`);
			if (!res.ok) {
				const err = await res.json().catch(() => ({ error: 'Failed to fetch repositories' }));
				throw new Error(err.error || 'Failed to load GitHub user repositories');
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
			if (storedGithubUsername) {
				fetchUserRepos(storedGithubUsername);
			}
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
		step = 3;
	}

	async function handleDeploy() {
		if (selectedType?.needsRepo && !repoUrl.trim()) {
			deployError = 'Repository URL is required';
			return;
		}

		isDeploying = true;
		deployError = null;

		try {
			let scanResult: ScanResult | null = null;

			// If repository URL provided, scan devpanel.yaml
			if (selectedType?.needsRepo && repoUrl.trim()) {
				try {
					scanResult = await scanRepo(repoUrl.trim(), appName.trim());
				} catch (e) {
					console.log('No devpanel.yaml found, using explicit UI form parameters');
				}
			}

			// Generate service configuration dynamically based on selected type
			let serviceType: 'web' | 'static' | 'database' | 'worker' = 'web';
			let image = '';

			switch (selectedType?.id) {
				case 'static':
					serviceType = 'static';
					break;
				case 'postgres':
					serviceType = 'database';
					image = `postgres:${dbVersion}-alpine`;
					if (!envVars['POSTGRES_PASSWORD']) envVars['POSTGRES_PASSWORD'] = 'postgres';
					if (!envVars['POSTGRES_DB']) envVars['POSTGRES_DB'] = appName.replace(/-/g, '_');
					break;
				case 'redis':
					serviceType = 'database';
					image = 'redis:7-alpine';
					break;
				case 'cron':
					serviceType = 'worker';
					break;
				default:
					serviceType = 'web';
					break;
			}

			const payloadServices = (scanResult?.services && scanResult.services.length > 0)
				? scanResult.services.map((s) => ({
						name: s.name,
						type: s.type,
						env_vars: { ...s.defaults?.env, ...envVars },
						port: s.defaults?.port || containerPort,
						custom_domain: '',
						auto_deploy: true,
						build_command: s.build?.command || buildCommand,
						start_command: s.deploy?.command || startCommand,
						instance_type: 'starter'
				  }))
				: [{
						name: appName.trim(),
						type: serviceType,
						image: image,
						env_vars: envVars,
						port: containerPort,
						custom_domain: '',
						auto_deploy: true,
						build_command: buildCommand,
						start_command: selectedType?.id === 'cron' ? cronCommand : startCommand,
						instance_type: 'starter'
				  }];

			const createRes = await createProject({
				app_name: appName.trim(),
				repo_url: selectedType?.needsRepo ? repoUrl.trim() : '',
				blueprint: scanResult?.blueprint || {
					version: '1.0',
					project: appName.trim(),
					services: {
						[appName.trim()]: {
							type: serviceType,
							image: image,
							source: { directory: '.', ref: 'main' },
							build: { engine: serviceType === 'static' ? 'static' : 'node', command: buildCommand, output_dir: publishDir },
							deploy: { port: containerPort, env: envVars }
						}
					}
				},
				services: payloadServices
			});

			const createdProjectId = createRes.blueprint?.id || appName.trim();
			await triggerDeploy(createdProjectId);
			goto(`/projects/${createdProjectId}`);
		} catch (err: any) {
			deployError = err.message || 'Failed to create and deploy service';
		} fontFinally: {
			isDeploying = false;
		}
	}

	onMount(() => {
		if (typeof window !== 'undefined') {
			storedGithubUsername = localStorage.getItem('devpnl_gh_username') || '';
		}
	});
</script>

<div class="p-6 md:p-10 max-w-5xl mx-auto w-full font-sans antialiased text-gray-900">
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
					{#if storedGithubUsername}
						Showing repositories for GitHub user <span class="font-semibold text-gray-800">{storedGithubUsername}</span>.
					{:else}
						No GitHub username configured in Settings. Enter a direct repository link below or set your username in Settings.
					{/if}
				</p>
			</div>

			{#if storedGithubUsername}
				<div class="bg-white border border-gray-200 rounded-xl overflow-hidden shadow-sm">
					<div class="p-4 border-b border-gray-200 bg-gray-50 flex items-center justify-between">
						<div class="flex items-center gap-2 text-sm font-medium text-gray-700">
							<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"/></svg>
							<span>GitHub Repositories</span>
							{#if isAuthenticated}
								<span class="ml-2 inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700 ring-1 ring-inset ring-green-600/20">Authenticated (Private Repos Enabled)</span>
							{:else}
								<span class="ml-2 inline-flex items-center rounded-md bg-yellow-50 px-2 py-1 text-xs font-medium text-yellow-800 ring-1 ring-inset ring-yellow-600/20">Public Only</span>
							{/if}
						</div>
						<span class="text-sm text-gray-500 font-mono">@{storedGithubUsername}</span>
					</div>

					{#if isFetchingRepos}
						<div class="p-8 text-center text-gray-500">
							<svg class="w-6 h-6 animate-spin mx-auto text-blue-600 mb-2" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
							<span>Fetching repositories for @{storedGithubUsername}...</span>
						</div>
					{:else if repoFetchError}
						<div class="p-6 text-center text-red-600 bg-red-50 text-sm">
							{repoFetchError}
						</div>
					{:else if userRepos.length === 0}
						<div class="p-8 text-center text-gray-500 text-sm">
							No repositories found for @{storedGithubUsername}. Enter a direct URL below.
						</div>
					{:else}
						<div class="divide-y divide-gray-100 max-h-80 overflow-y-auto">
							{#each userRepos as repo}
								<div class="p-4 flex items-center justify-between hover:bg-gray-50 transition-colors">
									<div class="flex items-center gap-3">
										<svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/></svg>
										<div>
											<div class="font-medium text-gray-900">{repo.full_name || repo.name}</div>
											<div class="text-xs text-gray-500">Updated {repo.updated}</div>
										</div>
									</div>
									<button
										type="button"
										onclick={() => handleRepoSelect(repo)}
										class="px-4 py-1.5 bg-white border border-gray-300 text-gray-700 text-sm font-medium rounded-lg hover:bg-gray-50 transition-colors shadow-sm"
									>
										Connect
									</button>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<!-- Fallback Direct Repo URL Box -->
			<div class="bg-white border border-gray-200 rounded-xl p-6 shadow-sm space-y-4">
				<h3 class="text-sm font-semibold text-gray-900 uppercase tracking-wider">Or Enter Direct Repository URL</h3>
				<div>
					<label for="directRepoUrl" class="block text-sm font-medium text-gray-700 mb-1.5">Git Repository URL</label>
					<input
						id="directRepoUrl"
						type="url"
						bind:value={repoUrl}
						placeholder="https://github.com/username/repository"
						class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 font-mono text-xs shadow-sm"
					/>
				</div>
				<div class="flex justify-end">
					<button
						type="button"
						onclick={() => (step = 3)}
						class="px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-medium text-sm rounded-lg transition-colors shadow-sm"
					>
						Continue with URL →
					</button>
				</div>
			</div>
		</div>
	{/if}

	<!-- Step 3: Dynamic Configure View per Service Type -->
	{#if step === 3 && selectedType}
		<div class="max-w-3xl mx-auto animate-in fade-in slide-in-from-bottom-4 duration-500 space-y-6">
			<div>
				<h2 class="text-2xl font-bold text-gray-900">Configure {selectedType.title}</h2>
				{#if selectedRepo}
					<p class="text-gray-500 mt-1 flex items-center gap-2 text-sm">
						Deploying from <span class="font-mono bg-gray-100 text-gray-700 px-1.5 py-0.5 rounded border border-gray-200">{selectedRepo.full_name || selectedRepo.name}</span>
					</p>
				{/if}
			</div>

			{#if deployError}
				<div class="p-4 rounded-xl bg-red-50 border border-red-200 text-red-700 text-sm">
					{deployError}
				</div>
			{/if}

			<div class="bg-white border border-gray-200 rounded-xl p-6 md:p-8 shadow-sm space-y-6">
				<!-- Common Fields -->
				<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
					<div>
						<label for="appDeployName" class="block text-sm font-medium text-gray-700 mb-1.5">Service Name</label>
						<input
							id="appDeployName"
							type="text"
							bind:value={appName}
							placeholder="e.g. my-service"
							class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 shadow-sm"
						/>
					</div>
					<div>
						<label for="appContainerPort" class="block text-sm font-medium text-gray-700 mb-1.5">Target Container Port</label>
						<input
							id="appContainerPort"
							type="number"
							bind:value={containerPort}
							class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 shadow-sm font-mono text-sm"
						/>
					</div>
				</div>

				{#if selectedType.needsRepo}
					<div>
						<label for="appRepoUrl" class="block text-sm font-medium text-gray-700 mb-1.5">Repository URL</label>
						<input
							id="appRepoUrl"
							type="text"
							bind:value={repoUrl}
							class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 shadow-sm font-mono text-xs"
						/>
					</div>
				{/if}

				<!-- Dynamic Specific Fields per Service Type -->
				{#if selectedType.id === 'web'}
					<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
						<div>
							<label for="webRuntime" class="block text-sm font-medium text-gray-700 mb-1.5">Runtime</label>
							<select id="webRuntime" bind:value={runtime} class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm">
								<option>Node.js</option>
								<option>Python</option>
								<option>Docker</option>
								<option>Go</option>
							</select>
						</div>
						<div>
							<label for="webBuildCmd" class="block text-sm font-medium text-gray-700 mb-1.5">Build Command</label>
							<input id="webBuildCmd" type="text" bind:value={buildCommand} class="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm" />
						</div>
						<div class="col-span-2">
							<label for="webStartCmd" class="block text-sm font-medium text-gray-700 mb-1.5">Start Command</label>
							<input id="webStartCmd" type="text" bind:value={startCommand} class="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm" />
						</div>
					</div>
				{:else if selectedType.id === 'static'}
					<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
						<div>
							<label for="staticBuildCmd" class="block text-sm font-medium text-gray-700 mb-1.5">Build Command</label>
							<input id="staticBuildCmd" type="text" bind:value={buildCommand} class="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm" />
						</div>
						<div>
							<label for="staticPublishDir" class="block text-sm font-medium text-gray-700 mb-1.5">Publish Directory</label>
							<input id="staticPublishDir" type="text" bind:value={publishDir} class="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm" />
						</div>
					</div>
				{:else if selectedType.id === 'postgres'}
					<div>
						<label for="pgVersion" class="block text-sm font-medium text-gray-700 mb-1.5">PostgreSQL Version</label>
						<select id="pgVersion" bind:value={dbVersion} class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm">
							<option value="15">15 (Latest Recommended)</option>
							<option value="14">14</option>
							<option value="13">13</option>
						</select>
					</div>
				{:else if selectedType.id === 'cron'}
					<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
						<div>
							<label for="cronSchedule" class="block text-sm font-medium text-gray-700 mb-1.5">Cron Schedule</label>
							<input id="cronSchedule" type="text" bind:value={cronSchedule} class="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm" />
						</div>
						<div>
							<label for="cronCmd" class="block text-sm font-medium text-gray-700 mb-1.5">Execute Command</label>
							<input id="cronCmd" type="text" bind:value={cronCommand} class="w-full font-mono text-sm bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 shadow-sm" />
						</div>
					</div>
				{/if}

				<!-- Environment Variables -->
				<div class="pt-4 border-t border-gray-100">
					<span class="block text-sm font-medium text-gray-900 mb-3">Environment Variables</span>
					<EnvVarEditor bind:envVars={envVars} />
				</div>
			</div>

			<div class="mt-8 flex justify-end gap-4">
				<button
					type="button"
					onclick={() => (step = selectedType.needsRepo ? 2 : 1)}
					class="px-5 py-2.5 text-gray-600 font-medium hover:bg-gray-100 rounded-lg transition-colors"
				>
					Back
				</button>
				<button
					type="button"
					onclick={handleDeploy}
					disabled={isDeploying}
					class="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors shadow-sm disabled:opacity-50 flex items-center gap-2"
				>
					{#if isDeploying}
						<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
						<span>Deploying Service...</span>
					{:else}
						<span>Create {selectedType.title}</span>
					{/if}
				</button>
			</div>
		</div>
	{/if}
</div>
