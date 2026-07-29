<script lang="ts">
	import { goto } from '$app/navigation';
	import { authState } from '$lib/auth';

	let password = $state('');
	let errorMsg = $state('');
	let isSubmitting = $state(false);

	async function handleSetup(e: Event) {
		e.preventDefault();
		if (!password || password.length < 6) {
			errorMsg = 'Password must be at least 6 characters long.';
			return;
		}

		isSubmitting = true;
		errorMsg = '';

		try {
			const res = await fetch('/api/auth/setup', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ password })
			});

			if (res.ok) {
				authState.update(s => ({ ...s, needsSetup: false, isAuthenticated: true }));
				goto('/');
			} else {
				const data = await res.json();
				errorMsg = data.error || 'Failed to setup password.';
			}
		} catch (err) {
			errorMsg = 'Network error occurred.';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="min-h-screen bg-[#09090b] flex flex-col justify-center py-12 sm:px-6 lg:px-8">
	<div class="sm:mx-auto sm:w-full sm:max-w-md">
		<div class="flex justify-center mb-6">
			<div class="w-12 h-12 bg-emerald-500/10 rounded-xl flex items-center justify-center border border-emerald-500/20 shadow-[0_0_15px_rgba(16,185,129,0.15)]">
				<svg class="w-6 h-6 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path></svg>
			</div>
		</div>
		<h2 class="mt-2 text-center text-3xl font-extrabold text-white tracking-tight">
			Welcome to DevPanel
		</h2>
		<p class="mt-2 text-center text-sm text-gray-400">
			It looks like this is your first time here. Let's set up your admin password.
		</p>
	</div>

	<div class="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
		<div class="bg-[#18181b] py-8 px-4 shadow-2xl sm:rounded-2xl sm:px-10 border border-white/5 ring-1 ring-white/10">
			<form class="space-y-6" onsubmit={handleSetup}>
				<div>
					<label for="password" class="block text-sm font-medium text-gray-300">
						Admin Password
					</label>
					<div class="mt-1">
						<input id="password" name="password" type="password" required bind:value={password}
							class="appearance-none block w-full px-3 py-2.5 border border-gray-700/50 rounded-xl shadow-sm placeholder-gray-500 focus:outline-none focus:ring-emerald-500 focus:border-emerald-500 sm:text-sm bg-black/40 text-white transition-colors"
							placeholder="Enter a secure password">
					</div>
				</div>

				{#if errorMsg}
					<div class="text-red-400 text-sm bg-red-400/10 p-3 rounded-lg border border-red-400/20">
						{errorMsg}
					</div>
				{/if}

				<div>
					<button type="submit" disabled={isSubmitting}
						class="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-xl shadow-sm text-sm font-medium text-black bg-emerald-500 hover:bg-emerald-400 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500 focus:ring-offset-[#09090b] transition-all disabled:opacity-50 disabled:cursor-not-allowed">
						{isSubmitting ? 'Saving...' : 'Set Password & Login'}
					</button>
				</div>
			</form>
		</div>
	</div>
</div>
