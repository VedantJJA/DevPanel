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
				errorMsg = data.error || 'Failed to set up password.';
			}
		} catch {
			errorMsg = 'Network error occurred.';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<svelte:head><title>Setup — DevPanel</title></svelte:head>

<div class="min-h-screen flex flex-col items-center justify-center relative overflow-hidden p-4" style="background-color: var(--background)">
	<!-- Background glow -->
	<div class="pointer-events-none absolute left-1/2 top-1/3 h-[600px] w-[600px] -translate-x-1/2 -translate-y-1/2 rounded-full opacity-20" style="background: radial-gradient(circle, color-mix(in oklch, var(--primary), transparent 50%), transparent 70%)"></div>

	<div class="relative z-10 w-full max-w-sm">
		<!-- Logo + Heading -->
		<div class="mb-8 flex flex-col items-center text-center">
			<div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl shadow-lg" style="background-color: var(--primary)">
				<span class="material-symbols-outlined text-white" style="font-size: 32px">lock</span>
			</div>
			<h1 class="text-3xl font-bold tracking-tight" style="color: var(--on-surface)">Welcome to DevPanel</h1>
			<p class="mt-2 text-sm" style="color: var(--on-surface-variant)">First time here? Set your admin password to get started.</p>
		</div>

		<!-- Steps indicator -->
		<div class="mb-6 flex items-center justify-center gap-2">
			<div class="flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold" style="background-color: var(--primary); color: var(--on-primary)">1</div>
			<div class="h-px w-8" style="background-color: var(--outline-variant)"></div>
			<div class="flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold" style="background-color: var(--surface-high); color: var(--on-surface-variant)">2</div>
			<span class="ml-1 text-xs" style="color: var(--on-surface-variant)">Set password → Launch</span>
		</div>

		<!-- Form Card -->
		<div class="card-surface p-8">
			<form onsubmit={handleSetup} class="space-y-5">
				<div>
					<label for="password" class="mb-1.5 block text-sm font-medium" style="color: var(--on-surface)">Admin Password</label>
					<input
						id="password"
						name="password"
						type="password"
						required
						bind:value={password}
						placeholder="Enter a secure password (min 6 chars)"
						class="w-full rounded-lg border px-4 py-3 text-sm outline-none transition-all"
						style="border-color: var(--outline-variant); background-color: var(--surface-lowest); color: var(--on-surface);"
						onfocus={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--primary)'; (e.currentTarget as HTMLElement).style.boxShadow = '0 0 0 3px color-mix(in oklch, var(--primary), transparent 80%)'; }}
						onblur={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--outline-variant)'; (e.currentTarget as HTMLElement).style.boxShadow = ''; }}
					/>
					<p class="mt-1.5 text-xs" style="color: var(--on-surface-variant)">Minimum 6 characters. This password protects all admin access.</p>
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
						Setting up…
					{:else}
						<span class="material-symbols-outlined" style="font-size: 18px">rocket_launch</span>
						Set Password & Launch
					{/if}
				</button>
			</form>
		</div>
	</div>
</div>
