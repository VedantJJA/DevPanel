<script lang="ts">
	interface Props {
		autoRefreshRateSec: number;
		githubUsername: string;
		actionLoading: string | null;
		onSetAutoRefresh: (rate: number) => void;
		onSetGithubUsername: (username: string) => void;
		onPruneSystem: () => void;
	}

	let {
		autoRefreshRateSec,
		githubUsername,
		actionLoading,
		onSetAutoRefresh,
		onSetGithubUsername,
		onPruneSystem
	}: Props = $props();

	let usernameInput = $state(githubUsername);
	let saveMessage = $state<string | null>(null);

	function handleSaveUsername(e: Event) {
		e.preventDefault();
		onSetGithubUsername(usernameInput.trim());
		saveMessage = 'GitHub username saved successfully!';
		setTimeout(() => (saveMessage = null), 3000);
	}
</script>

<div class="p-6 md:p-10 max-w-5xl mx-auto w-full font-sans text-gray-900">
	<div class="mb-8">
		<h1 class="text-2xl font-semibold text-gray-900 tracking-tight">System Settings</h1>
		<p class="text-gray-500 mt-1">Configure global platform preferences and GitHub integrations.</p>
	</div>

	{#if saveMessage}
		<div class="mb-6 p-4 rounded-xl bg-green-50 border border-green-200 text-green-700 text-sm flex items-center justify-between">
			<span>{saveMessage}</span>
			<button type="button" onclick={() => (saveMessage = null)} class="text-green-700 font-bold">✕</button>
		</div>
	{/if}

	<div class="space-y-8">
		<!-- GitHub Integration Settings -->
		<section class="bg-white border border-gray-200 rounded-xl p-6 md:p-8 shadow-sm space-y-4">
			<div>
				<h3 class="text-lg font-semibold text-gray-900 flex items-center gap-2">
					<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"/></svg>
					GitHub Account Integration
				</h3>
				<p class="text-sm text-gray-500 mt-1">Provide your GitHub username to automatically populate your public repositories in the service creation wizard.</p>
			</div>

			<form onsubmit={handleSaveUsername} class="space-y-4 max-w-xl">
				<div>
					<label for="ghUsername" class="block text-sm font-medium text-gray-700 mb-1.5">GitHub Username</label>
					<input
						id="ghUsername"
						type="text"
						bind:value={usernameInput}
						placeholder="e.g. VedantJJA"
						class="w-full bg-white border border-gray-300 rounded-lg px-4 py-2.5 text-gray-900 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 shadow-sm"
					/>
				</div>
				<button
					type="submit"
					class="bg-blue-600 hover:bg-blue-700 text-white px-5 py-2.5 rounded-lg text-sm font-medium transition-colors shadow-sm"
				>
					Save Username
				</button>
			</form>
		</section>

		<!-- Refresh & Performance Settings -->
		<section class="bg-white border border-gray-200 rounded-xl p-6 md:p-8 shadow-sm space-y-4">
			<div>
				<h3 class="text-lg font-semibold text-gray-900">Dashboard Telemetry Refresh</h3>
				<p class="text-sm text-gray-500 mt-1">Set how frequently Docker container stats update on screen.</p>
			</div>

			<div class="flex items-center gap-3">
				{#each [2, 5, 10, 30] as rate}
					<button
						type="button"
						onclick={() => onSetAutoRefresh(rate)}
						class="px-4 py-2 rounded-lg text-sm font-medium transition-colors border {autoRefreshRateSec === rate
							? 'bg-blue-50 text-blue-700 border-blue-200 shadow-sm'
							: 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'}"
					>
						{rate} Seconds
					</button>
				{/each}
			</div>
		</section>

		<!-- System Prune Danger Zone -->
		<section class="bg-white border border-red-200 rounded-xl p-6 md:p-8 shadow-sm space-y-4">
			<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
				<div>
					<h3 class="text-lg font-semibold text-red-600">Docker System Maintenance</h3>
					<p class="text-sm text-gray-500 mt-1">Prune stopped containers, unused networks, and dangling build caches.</p>
				</div>
				<button
					type="button"
					onclick={onPruneSystem}
					disabled={actionLoading === 'prune'}
					class="bg-red-50 hover:bg-red-100 text-red-600 border border-red-200 px-5 py-2.5 rounded-lg text-sm font-medium transition-colors whitespace-nowrap disabled:opacity-50"
				>
					{actionLoading === 'prune' ? 'Pruning System...' : 'Prune Docker System'}
				</button>
			</div>
		</section>
	</div>
</div>
