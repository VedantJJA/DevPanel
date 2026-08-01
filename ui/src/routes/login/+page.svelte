<script lang="ts">
	import { goto } from '$app/navigation';
	import { authState } from '$lib/auth';

	let password = $state('');
	let errorMsg = $state('');
	let isSubmitting = $state(false);

	async function handleLogin(e: Event) {
		e.preventDefault();
		if (!password) { errorMsg = 'Password is required.'; return; }
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
		} catch {
			errorMsg = 'Network error occurred.';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<svelte:head><title>Sign in — DevPanel</title></svelte:head>

<div class="min-h-screen flex flex-col items-center justify-center relative overflow-hidden p-4" style="background-color: var(--background)">
	<!-- Background glow -->
	<div class="pointer-events-none absolute left-1/2 top-1/3 h-[600px] w-[600px] -translate-x-1/2 -translate-y-1/2 rounded-full opacity-30" style="background: radial-gradient(circle, color-mix(in oklch, var(--primary), transparent 60%), transparent 70%)"></div>

	<!-- Card -->
	<div class="relative z-10 w-full max-w-sm">
		<!-- Logo -->
		<div class="mb-8 flex flex-col items-center text-center">
			<div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl shadow-lg" style="background-color: var(--primary)">
				<span class="material-symbols-outlined text-white" style="font-size: 32px">dns</span>
			</div>
			<h1 class="text-3xl font-bold tracking-tight" style="color: var(--on-surface)">DevPanel</h1>
			<p class="mt-2 text-sm" style="color: var(--on-surface-variant)">Enter your master password to access the dashboard</p>
		</div>

		<!-- Form Card -->
		<div class="card-surface p-8">
			<form onsubmit={handleLogin} class="space-y-5">
				<div>
					<label for="password" class="mb-1.5 block text-sm font-medium" style="color: var(--on-surface)">Password</label>
					<input
						id="password"
						name="password"
						type="password"
						required
						bind:value={password}
						placeholder="••••••••"
						class="w-full rounded-lg border px-4 py-3 text-sm outline-none transition-all"
						style="border-color: var(--outline-variant); background-color: var(--surface-lowest); color: var(--on-surface);"
						onfocus={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--primary)'; (e.currentTarget as HTMLElement).style.boxShadow = '0 0 0 3px color-mix(in oklch, var(--primary), transparent 80%)'; }}
						onblur={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--outline-variant)'; (e.currentTarget as HTMLElement).style.boxShadow = ''; }}
					/>
				</div>

				{#if errorMsg}
					<div class="flex items-center gap-2 rounded-lg border px-4 py-3 text-sm" style="background-color: var(--error-container); border-color: var(--error); color: var(--error)">
						<span class="material-symbols-outlined" style="font-size: 18px">error</span>
						{errorMsg}
					</div>
				{/if}

				<button
					type="submit"
					disabled={isSubmitting}
					class="flex w-full items-center justify-center gap-2 rounded-lg py-3 text-sm font-semibold transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
					style="background-color: var(--primary); color: var(--on-primary);"
				>
					{#if isSubmitting}
						<span class="material-symbols-outlined animate-spin" style="font-size: 18px">refresh</span>
						Authenticating…
					{:else}
						<span class="material-symbols-outlined" style="font-size: 18px">login</span>
						Sign In
					{/if}
				</button>
			</form>
		</div>

		<p class="mt-6 text-center text-xs" style="color: var(--on-surface-variant)">
			Self-hosted PaaS · Secure access required
		</p>
	</div>
</div>
