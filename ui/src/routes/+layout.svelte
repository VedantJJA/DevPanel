<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authState, checkAuthStatus } from '$lib/auth';
	import { loadRoutingConfig } from '$lib/stores/routing';

	let { children } = $props();

	onMount(async () => {
		await loadRoutingConfig();
		const data = await checkAuthStatus();
		if (data) {
			if (data.needs_setup && $page.url.pathname !== '/setup') {
				goto('/setup');
			} else if (!data.needs_setup && !data.authenticated && $page.url.pathname !== '/login') {
				goto('/login');
			}
		}
	});

	$effect(() => {
		if (!$authState.isLoading) {
			if ($authState.needsSetup && $page.url.pathname !== '/setup') {
				goto('/setup');
			} else if (!$authState.needsSetup && !$authState.isAuthenticated && $page.url.pathname !== '/login') {
				goto('/login');
			}
		}
	});
</script>

<svelte:head>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>DevPanel — Deploy &amp; Manage Without SSH</title>
	<meta name="description" content="One control plane for your cloud VM: deploy projects, run containers, route traffic and watch metrics without touching SSH." />
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Hanken+Grotesk:wght@600;700&family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@500&display=swap" />
	<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@20..48,100..700,0..1,-50..200&display=swap" />
</svelte:head>

{#if $authState.isLoading}
	<div class="h-screen w-screen flex items-center justify-center" style="background-color: var(--background)">
		<div class="flex flex-col items-center gap-4">
			<span class="material-symbols-outlined animate-spin" style="font-size:32px; color: var(--primary)">refresh</span>
			<p style="color: var(--on-surface-variant); font-size: 14px;">Loading DevPanel…</p>
		</div>
	</div>
{:else}
	{@render children()}
{/if}
