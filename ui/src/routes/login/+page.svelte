<script lang="ts">
	import { goto } from '$app/navigation';
	import { authState } from '$lib/auth';

	let password = $state('');
	let errorMsg = $state('');
	let isSubmitting = $state(false);

	async function handleLogin(e: Event) {
		e.preventDefault();
		if (!password) {
			errorMsg = 'Password is required.';
			return;
		}

		isSubmitting = true;
		errorMsg = '';

		try {
			const res = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ password })
			});

			if (res.ok) {
				authState.update(s => ({ ...s, isAuthenticated: true }));
				goto('/');
			} else {
				const data = await res.json();
				errorMsg = data.error || 'Invalid password.';
			}
		} catch (err) {
			errorMsg = 'Network error occurred.';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<div class="min-h-screen bg-[#09090b] flex flex-col justify-center py-12 sm:px-6 lg:px-8 relative overflow-hidden">
	<!-- Background glow effect -->
	<div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] bg-emerald-500/10 rounded-full blur-[100px] pointer-events-none"></div>

	<div class="sm:mx-auto sm:w-full sm:max-w-md relative z-10">
		<div class="flex justify-center mb-6">
			<div class="w-16 h-16 bg-[#18181b] rounded-2xl flex items-center justify-center border border-white/10 shadow-2xl relative group">
				<div class="absolute inset-0 bg-emerald-500/20 rounded-2xl blur-md opacity-0 group-hover:opacity-100 transition-opacity"></div>
				<svg class="w-8 h-8 text-emerald-400 relative z-10" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path></svg>
			</div>
		</div>
		<h2 class="text-center text-4xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-white to-gray-400 tracking-tight">
			Sign in to DevPanel
		</h2>
		<p class="mt-3 text-center text-sm text-gray-400">
			Enter your master password to access the dashboard
		</p>
	</div>

	<div class="mt-8 sm:mx-auto sm:w-full sm:max-w-md relative z-10">
		<div class="bg-[#18181b]/80 backdrop-blur-xl py-8 px-4 shadow-2xl sm:rounded-2xl sm:px-10 border border-white/5 ring-1 ring-white/10">
			<form class="space-y-6" onsubmit={handleLogin}>
				<div>
					<label for="password" class="block text-sm font-medium text-gray-300">
						Password
					</label>
					<div class="mt-2">
						<input id="password" name="password" type="password" required bind:value={password}
							class="appearance-none block w-full px-4 py-3 border border-gray-700/50 rounded-xl shadow-sm placeholder-gray-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/50 focus:border-emerald-500 sm:text-sm bg-black/40 text-white transition-all"
							placeholder="••••••••">
					</div>
				</div>

				{#if errorMsg}
					<div class="text-red-400 text-sm bg-red-400/10 p-3 rounded-lg border border-red-400/20 flex items-center gap-2">
						<svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
						{errorMsg}
					</div>
				{/if}

				<div>
					<button type="submit" disabled={isSubmitting}
						class="w-full flex justify-center py-3 px-4 border border-transparent rounded-xl shadow-[0_0_20px_rgba(16,185,129,0.2)] text-sm font-semibold text-black bg-emerald-500 hover:bg-emerald-400 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-emerald-500 focus:ring-offset-[#09090b] transition-all disabled:opacity-50 disabled:cursor-not-allowed hover:shadow-[0_0_30px_rgba(16,185,129,0.4)] hover:-translate-y-0.5 active:translate-y-0">
						{isSubmitting ? 'Authenticating...' : 'Sign In'}
					</button>
				</div>
			</form>
		</div>
	</div>
</div>
