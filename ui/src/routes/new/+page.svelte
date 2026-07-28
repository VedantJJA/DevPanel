<script lang="ts">
	import { goto } from '$app/navigation';
	import Stepper from '$lib/components/Stepper.svelte';
	import EnvVarEditor from '$lib/components/EnvVarEditor.svelte';
	import Terminal from '$lib/components/Terminal.svelte';
	import { scanRepo, createProject, triggerDeploy } from '$lib/api';
	import type { ScanResult, ScanService } from '$lib/types';

	let step = $state<1 | 2 | 3>(1);

	// Step 1 State
	let appName = $state('test-repo');
	let repoUrl = $state('https://github.com/VedantJJA/DevPtestrepo');
	let isScanning = $state(false);
	let scanResult = $state<ScanResult | null>(null);
	let scanError = $state<string | null>(null);

	// Step 2 State
	let services = $state<ScanService[]>([]);
	let activeServiceIdx = $state(0);
	let isCreating = $state(false);
	let createError = $state<string | null>(null);
	let createdProjectId = $state<string | null>(null);

	// Step 3 State
	let isDeploying = $state(false);
	let deployFinished = $state(false);

	async function handleScanSubmit(e: Event) {
		e.preventDefault();
		if (!repoUrl.trim()) return;

		isScanning = true;
		scanError = null;
		scanResult = null;

		try {
			const res = await scanRepo(repoUrl.trim(), appName.trim());
			scanResult = res;

			if (res.errors && res.errors.length > 0) {
				scanError = res.errors[0];
			} else {
				services = res.services || [];
			}
		} catch (err: any) {
			scanError = err.message || 'Failed to scan repository blueprint';
		} finally {
			isScanning = false;
		}
	}

	function goToStep2() {
		if (scanResult && (!scanResult.errors || scanResult.errors.length === 0)) {
			step = 2;
		}
	}

	async function handleCreateAndDeploy() {
		if (!scanResult) return;
		isCreating = true;
		createError = null;

		try {
			const payloadServices = services.map((s) => ({
				name: s.name,
				type: s.type,
				env_vars: s.defaults.env,
				port: s.defaults.port,
				custom_domain: '',
				auto_deploy: true,
				build_command: s.build?.command || '',
				start_command: s.deploy?.command || '',
				instance_type: 'free'
			}));

			const res = await createProject({
				app_name: scanResult.project || appName,
				repo_url: repoUrl.trim(),
				blueprint: scanResult.blueprint,
				services: payloadServices
			});

			createdProjectId = res.blueprint.id;
			step = 3;

			// Trigger async deployment
			await triggerDeploy(createdProjectId);
			isDeploying = true;
		} catch (err: any) {
			createError = err.message || 'Failed to initialize project deployment';
		} finally {
			isCreating = false;
		}
	}
</script>

<div class="min-h-screen bg-neutral-950 text-neutral-100 p-6 md:p-12 font-sans antialiased">
	<div class="max-w-5xl mx-auto space-y-8">
		<!-- Header -->
		<div class="text-center space-y-2">
			<h1 class="text-3xl font-bold tracking-tight text-neutral-100 sm:text-4xl">New Deployment Wizard</h1>
			<p class="text-xs text-neutral-400">Deploy multi-service monorepos and static web applications in 3 quick steps.</p>
		</div>

		<!-- Stepper Progress Bar -->
		<Stepper currentStep={step} />

		<!-- STEP 1: Repository Select & Scan -->
		{#if step === 1}
			<div class="max-w-2xl mx-auto space-y-6">
				<form onsubmit={handleScanSubmit} class="p-8 rounded-3xl bg-neutral-900/60 border border-neutral-800 space-y-6 shadow-2xl backdrop-blur-md">
					<div class="space-y-2">
						<label for="appName" class="block text-xs font-semibold uppercase tracking-wider text-neutral-300">
							App / Project Name
						</label>
						<input
							id="appName"
							type="text"
							bind:value={appName}
							placeholder="e.g. my-startup"
							class="w-full px-4 py-3 bg-neutral-950 border border-neutral-800 rounded-xl text-sm text-neutral-100 focus:outline-none focus:border-emerald-500 font-mono"
							required
						/>
					</div>

					<div class="space-y-2">
						<label for="repoUrl" class="block text-xs font-semibold uppercase tracking-wider text-neutral-300">
							GitHub Repository URL
						</label>
						<input
							id="repoUrl"
							type="url"
							bind:value={repoUrl}
							placeholder="https://github.com/user/repo"
							class="w-full px-4 py-3 bg-neutral-950 border border-neutral-800 rounded-xl text-sm text-neutral-100 focus:outline-none focus:border-emerald-500 font-mono"
							required
						/>
					</div>

					<button
						type="submit"
						disabled={isScanning}
						class="w-full py-3.5 px-6 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-sm flex items-center justify-center gap-3 transition-all shadow-lg disabled:opacity-50"
					>
						{#if isScanning}
							<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
							<span>Scanning devpanel.yaml...</span>
						{:else}
							<span>Scan & Parse devpanel.yaml →</span>
						{/if}
					</button>
				</form>

				<!-- Scan Result Banner -->
				{#if scanResult}
					<div class="p-6 rounded-3xl bg-neutral-900/80 border border-neutral-800 space-y-4 shadow-xl">
						<div class="flex items-center justify-between">
							<h3 class="font-bold text-base text-neutral-100 font-mono">Scan Results for {scanResult.project}</h3>
							<span class="px-2.5 py-0.5 rounded-full text-xs font-semibold font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
								{scanResult.services.length} Services Detected
							</span>
						</div>

						<!-- Errors List -->
						{#if scanResult.errors && scanResult.errors.length > 0}
							<div class="p-4 rounded-2xl bg-rose-950/60 border border-rose-800/80 text-rose-200 text-xs space-y-1 font-mono">
								<div class="font-bold text-rose-400 uppercase tracking-wider">Validation Errors (Blocked):</div>
								{#each scanResult.errors as errStr}
									<div class="flex items-start gap-2">
										<span>❌</span>
										<span>{errStr}</span>
									</div>
								{/each}
							</div>
						{/if}

						<!-- Warnings List -->
						{#if scanResult.warnings && scanResult.warnings.length > 0}
							<div class="p-4 rounded-2xl bg-amber-950/40 border border-amber-800/60 text-amber-200 text-xs space-y-1 font-mono">
								<div class="font-bold text-amber-400 uppercase tracking-wider">Warnings:</div>
								{#each scanResult.warnings as warnStr}
									<div class="flex items-start gap-2">
										<span>⚠️</span>
										<span>{warnStr}</span>
									</div>
								{/each}
							</div>
						{/if}

						<!-- Action Next Button -->
						{#if !scanResult.errors || scanResult.errors.length === 0}
							<div class="pt-2 flex justify-end">
								<button
									onclick={goToStep2}
									class="px-6 py-3 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs flex items-center gap-2 transition-all shadow-lg"
								>
									<span>Next: Configure Services →</span>
								</button>
							</div>
						{/if}
					</div>
				{/if}
			</div>
		{/if}

		<!-- STEP 2: Configure Per-Service Settings -->
		{#if step === 2 && scanResult}
			<div class="space-y-6">
				<!-- Service Selector Tabs -->
				<div class="flex items-center gap-2 border-b border-neutral-800 pb-3 overflow-x-auto">
					{#each services as svc, idx}
						<button
							onclick={() => (activeServiceIdx = idx)}
							class="px-4 py-2 rounded-xl text-xs font-mono font-bold transition-all flex items-center gap-2 {activeServiceIdx === idx
								? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
								: 'text-neutral-400 hover:text-neutral-200'}"
						>
							<span>{svc.name}</span>
							<span class="px-1.5 py-0.5 rounded text-[10px] uppercase font-bold bg-neutral-900 border border-neutral-800">{svc.type}</span>
						</button>
					{/each}
				</div>

				<!-- Active Service Settings Card -->
				{#if services[activeServiceIdx]}
					{@const currentSvc = services[activeServiceIdx]}
					<div class="p-6 rounded-3xl bg-neutral-900/60 border border-neutral-800 space-y-6 shadow-xl">
						<div class="flex items-center justify-between border-b border-neutral-800 pb-4">
							<div>
								<h2 class="text-xl font-bold text-neutral-100 font-mono">{currentSvc.name}</h2>
								<p class="text-xs text-neutral-400 font-mono">Service Type: {currentSvc.type}</p>
							</div>
							<span class="px-3 py-1 rounded-full text-xs font-semibold font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
								{currentSvc.type}
							</span>
						</div>

						<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
							<!-- Port Configuration -->
							<div class="space-y-2">
								<label for="svcPort" class="block text-xs font-semibold uppercase text-neutral-400">Target Container Port</label>
								<input
									id="svcPort"
									type="number"
									bind:value={currentSvc.defaults.port}
									class="w-full px-4 py-2.5 bg-neutral-950 border border-neutral-800 rounded-xl text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
								/>
							</div>

							<!-- Custom Domain -->
							<div class="space-y-2">
								<label for="svcDomain" class="block text-xs font-semibold uppercase text-neutral-400">Custom Domain (Optional)</label>
								<input
									id="svcDomain"
									type="text"
									placeholder="app.example.com"
									class="w-full px-4 py-2.5 bg-neutral-950 border border-neutral-800 rounded-xl text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
								/>
							</div>
						</div>

						<!-- Environment Variables Editor -->
						<div class="space-y-2 pt-2 border-t border-neutral-800/80">
							<span class="block text-xs font-semibold uppercase text-neutral-400">Environment Variables</span>
							<EnvVarEditor bind:envVars={currentSvc.defaults.env} />
						</div>
					</div>
				{/if}

				<!-- Action Buttons -->
				<div class="flex items-center justify-between pt-4 border-t border-neutral-800">
					<button
						onclick={() => (step = 1)}
						class="px-4 py-2 rounded-xl bg-neutral-900 border border-neutral-800 text-neutral-400 hover:text-neutral-200 text-xs font-semibold"
					>
						← Back to Scan
					</button>

					<button
						onclick={handleCreateAndDeploy}
						disabled={isCreating}
						class="px-8 py-3.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs flex items-center gap-2 transition-all shadow-xl disabled:opacity-50"
					>
						{#if isCreating}
							<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
							<span>Initializing Pipeline...</span>
						{:else}
							<span>Deploy Application Stack →</span>
						{/if}
					</button>
				</div>
			</div>
		{/if}

		<!-- STEP 3: Live Deploy SSE Terminal -->
		{#if step === 3 && createdProjectId}
			<div class="space-y-6">
				<div class="h-[500px]">
					<Terminal projectId={createdProjectId} title="Step 3: Real-Time Build & Deploy Stream" />
				</div>

				<div class="flex justify-end pt-4">
					<a
						href="/projects/{createdProjectId}"
						class="px-8 py-3.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs flex items-center gap-2 transition-all shadow-xl"
					>
						<span>Open Project Dashboard →</span>
					</a>
				</div>
			</div>
		{/if}
	</div>
</div>
