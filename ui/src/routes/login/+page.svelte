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

<div class="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8 relative overflow-hidden">
	<!-- Background graphic -->
	<div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] bg-blue-100/50 rounded-full blur-[100px] pointer-events-none"></div>

	<div class="sm:mx-auto sm:w-full sm:max-w-md relative z-10">
		<div class="flex justify-center mb-6">
			<div class="w-16 h-16 bg-white rounded-2xl flex items-center justify-center border border-gray-200 shadow-xl relative group">
				<div class="absolute inset-0 bg-blue-500/10 rounded-2xl blur-md opacity-0 group-hover:opacity-100 transition-opacity"></div>
				<svg class="w-8 h-8 text-blue-600 relative z-10" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"></path></svg>
			</div>
		</div>
		<h2 class="text-center text-3xl font-extrabold text-gray-900 tracking-tight">
			Sign in to DevPanel
		</h2>
		<p class="mt-2 text-center text-sm text-gray-500">
			Enter your master password to access the dashboard
		</p>
	</div>

	<div class="mt-8 sm:mx-auto sm:w-full sm:max-w-md relative z-10">
		<div class="bg-white py-8 px-4 shadow-xl sm:rounded-2xl sm:px-10 border border-gray-200">
			<form class="space-y-6" onsubmit={handleLogin}>
				<div>
					<label for="password" class="block text-sm font-medium text-gray-700">
						Password
					</label>
					<div class="mt-2">
						<input id="password" name="password" type="password" required bind:value={password}
							class="appearance-none block w-full px-4 py-3 border border-gray-300 rounded-xl shadow-sm placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm bg-white text-gray-900 transition-colors"
							placeholder="••••••••">
					</div>
				</div>

				{#if errorMsg}
					<div class="text-red-700 text-sm bg-red-50 p-3 rounded-lg border border-red-200 flex items-center gap-2">
						<svg class="w-4 h-4 shrink-0 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
						{errorMsg}
					</div>
				{/if}

				<div>
					<button type="submit" disabled={isSubmitting}
						class="w-full flex justify-center py-3 px-4 border border-transparent rounded-xl shadow-sm text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-all disabled:opacity-50 disabled:cursor-not-allowed hover:-translate-y-0.5 active:translate-y-0">
						{isSubmitting ? 'Authenticating...' : 'Sign In'}
					</button>
				</div>
			</form>
		</div>
	</div>
</div>
