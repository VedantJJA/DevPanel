<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authState, checkAuthStatus } from '$lib/auth';

	let { children } = $props();
	
	onMount(async () => {
		const data = await checkAuthStatus();
		if (data) {
			if (data.needs_setup && $page.url.pathname !== '/setup') {
				goto('/setup');
			} else if (!data.needs_setup && !data.authenticated && $page.url.pathname !== '/login') {
				goto('/login');
			}
		}
	});

	// Reactively redirect if auth state changes (e.g. logout)
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

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

{#if $authState.isLoading}
	<div class="h-screen w-screen flex items-center justify-center bg-[#09090b]">
		<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-emerald-500"></div>
	</div>
{:else}
	{@render children()}
{/if}
