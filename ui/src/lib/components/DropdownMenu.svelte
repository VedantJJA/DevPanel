<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	interface MenuItem {
		label?: string;
		icon?: any;
		danger?: boolean;
		divider?: boolean;
		onClick?: () => void;
	}

	interface Props {
		items: MenuItem[];
		right?: boolean;
		trigger: any;
	}

	let { items, right = false, trigger }: Props = $props();

	let isOpen = $state(false);
	let menuRef: HTMLDivElement | null = $state(null);

	function toggle(e: MouseEvent) {
		e.stopPropagation();
		isOpen = !isOpen;
	}

	function close() {
		isOpen = false;
	}

	function handleItemClick(item: MenuItem, e: MouseEvent) {
		e.stopPropagation();
		isOpen = false;
		if (item.onClick) {
			item.onClick();
		}
	}

	function handleWindowClick(e: MouseEvent) {
		if (menuRef && !menuRef.contains(e.target as Node)) {
			close();
		}
	}

	onMount(() => {
		window.addEventListener('click', handleWindowClick);
	});

	onDestroy(() => {
		if (typeof window !== 'undefined') {
			window.removeEventListener('click', handleWindowClick);
		}
	});
</script>

<div class="relative inline-block text-left" bind:this={menuRef}>
	<div onclick={toggle} role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && toggle(e as any)}>
		{@render trigger()}
	</div>

	{#if isOpen}
		<div
			class="absolute z-50 mt-2 w-48 rounded-xl bg-white border border-gray-200 shadow-lg py-1 animate-in fade-in zoom-in-95 duration-100 {right
				? 'right-0 origin-top-right'
				: 'left-0 origin-top-left'}"
		>
			{#each items as item, idx}
				{#if item.divider}
					<div class="h-px bg-gray-100 my-1"></div>
				{:else}
					<button
						type="button"
						onclick={(e) => handleItemClick(item, e)}
						class="w-full text-left px-4 py-2 text-sm font-medium flex items-center gap-2.5 transition-colors {item.danger
							? 'text-red-600 hover:bg-red-50'
							: 'text-gray-700 hover:bg-gray-50'}"
					>
						{#if item.icon}
							{@const Icon = item.icon}
							<Icon class="w-4 h-4 text-gray-400 group-hover:text-gray-600" />
						{/if}
						<span>{item.label}</span>
					</button>
				{/if}
			{/each}
		</div>
	{/if}
</div>
