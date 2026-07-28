<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Blueprint } from '$lib/types';

	let appName = $state('');
	let repoUrl = $state('');
	let loading = $state(false);

	let alertState = $state<{
		type: 'success' | 'error';
		message: string;
		blueprint?: Blueprint;
	} | null>(null);

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!appName.trim() || !repoUrl.trim()) {
			alertState = {
				type: 'error',
				message: 'Please fill in both App Name and GitHub Repository URL.'
			};
			return;
		}

		loading = true;
		alertState = null;

		try {
			const res = await fetch('/api/blueprints/validate', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					app_name: appName.trim(),
					repo_url: repoUrl.trim()
				})
			});

			const data = await res.json();

			if (!res.ok || data.error) {
				alertState = {
					type: 'error',
					message: data.error || 'Failed to validate repository blueprint.'
				};
			} else {
				alertState = {
					type: 'success',
					message: data.message || 'Blueprint validated successfully!',
					blueprint: data.blueprint
				};

				// Store validated blueprint payload for the Setup Review Page
				if (typeof window !== 'undefined') {
					sessionStorage.setItem('devpanel_setup_data', JSON.stringify({
						appName: appName.trim(),
						repoUrl: repoUrl.trim(),
						blueprint: data.blueprint
					}));
				}

				// Redirect to Setup Review page after 1 second
				setTimeout(() => {
					goto('/blueprints/setup');
				}, 1200);
			}
		} catch (err: any) {
			alertState = {
				type: 'error',
				message: `Network error: ${err.message || 'Unable to connect to DevPanel server.'}`
			};
		} finally {
			loading = false;
		}
	}

	function proceedToSetup() {
		goto('/blueprints/setup');
	}
</script>

<div class="min-h-screen bg-neutral-950 text-neutral-100 p-6 md:p-12 font-sans antialiased flex flex-col items-center justify-center">
	<div class="w-full max-w-2xl space-y-8">
		<!-- Header -->
		<div class="space-y-2 text-center">
			<div class="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs font-mono">
				<span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
				New Blueprint Deployment
			</div>
			<h1 class="text-3xl font-bold tracking-tight text-neutral-100 sm:text-4xl">Deploy Application</h1>
			<p class="text-sm text-neutral-400 max-w-md mx-auto">
				Connect your GitHub repository containing a <code class="text-emerald-400 font-mono">devpanel.yaml</code> blueprint to automatically index and provision your stack.
			</p>
		</div>

		<!-- Feedback Toast / Alert Banner -->
		{#if alertState}
			<div
				class="p-5 rounded-2xl border transition-all text-sm shadow-xl flex items-start gap-3.5 {alertState.type === 'error'
					? 'bg-rose-950/60 border-rose-800/80 text-rose-200'
					: 'bg-emerald-950/60 border-emerald-800/80 text-emerald-200'}"
			>
				<div class="mt-0.5 shrink-0">
					{#if alertState.type === 'error'}
						<svg class="w-5 h-5 text-rose-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
					{:else}
						<svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
					{/if}
				</div>

				<div class="space-y-3 flex-1">
					<p class="font-bold text-base leading-relaxed">{alertState.message}</p>

					{#if alertState.type === 'success'}
						<p class="text-xs text-emerald-300">Redirecting to Setup Review page in 1 second...</p>

						{#if alertState.blueprint}
							<div class="mt-2 p-3 rounded-xl bg-neutral-900/90 border border-neutral-800 text-xs font-mono space-y-1.5 text-neutral-300">
								<div class="text-emerald-400 font-bold">Verified Services:</div>
								{#each Object.entries(alertState.blueprint.services || {}) as [svcName, svcCfg]}
									<div class="flex items-center justify-between border-b border-neutral-800/60 py-1">
										<span class="text-neutral-200 font-semibold">{svcName}</span>
										<span class="text-neutral-400">type: {svcCfg.type} | {svcCfg.image ? `image: ${svcCfg.image}` : `dir: ${svcCfg.source?.directory || './'}`}</span>
									</div>
								{/each}
							</div>
						{/if}

						<div class="pt-2">
							<button
								onclick={proceedToSetup}
								class="px-4 py-2 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs flex items-center gap-2 transition-all shadow-md"
							>
								<span>Proceed to Setup Review →</span>
							</button>
						</div>
					{/if}
				</div>

				<button
					onclick={() => (alertState = null)}
					class="text-neutral-400 hover:text-neutral-200 text-xs font-bold p-1"
				>
					✕
				</button>
			</div>
		{/if}

		<!-- Form Card -->
		<form
			onsubmit={handleSubmit}
			class="p-8 rounded-3xl bg-neutral-900/60 border border-neutral-800 space-y-6 shadow-2xl backdrop-blur-md"
		>
			<div class="space-y-2">
				<label for="appName" class="block text-xs font-semibold uppercase tracking-wider text-neutral-300">
					App Name
				</label>
				<input
					id="appName"
					type="text"
					bind:value={appName}
					placeholder="e.g., test-repo"
					disabled={loading}
					class="w-full px-4 py-3 bg-neutral-950 border border-neutral-800 rounded-xl text-sm text-neutral-100 placeholder-neutral-500 focus:outline-none focus:border-emerald-500/60 transition-all disabled:opacity-50"
					required
				/>
			</div>

			<div class="space-y-2">
				<label for="repoUrl" class="block text-xs font-semibold uppercase tracking-wider text-neutral-300">
					GitHub Repository URL
				</label>
				<div class="relative">
					<input
						id="repoUrl"
						type="url"
						bind:value={repoUrl}
						placeholder="https://github.com/username/my-monorepo.git"
						disabled={loading}
						class="w-full pl-11 pr-4 py-3 bg-neutral-950 border border-neutral-800 rounded-xl text-sm text-neutral-100 placeholder-neutral-500 focus:outline-none focus:border-emerald-500/60 transition-all disabled:opacity-50"
						required
					/>
					<svg class="w-5 h-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-neutral-500" fill="currentColor" viewBox="0 0 24 24"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"/></svg>
				</div>
			</div>

			<button
				type="submit"
				disabled={loading}
				class="w-full py-3.5 px-6 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-sm flex items-center justify-center gap-3 transition-all shadow-lg shadow-emerald-950/50 disabled:opacity-50"
			>
				{#if loading}
					<svg class="w-4 h-4 animate-spin text-white" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path></svg>
					<span>Indexing Repository...</span>
				{:else}
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3"/></svg>
					<span>Index & Validate Blueprint</span>
				{/if}
			</button>
		</form>

		<div class="text-center">
			<a href="/" class="text-xs font-medium text-neutral-400 hover:text-emerald-400 transition-colors">
				← Back to Dashboard
			</a>
		</div>
	</div>
</div>
