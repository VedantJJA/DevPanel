<script lang="ts">
	import type { ErrorModalState } from '$lib/types';

	interface Props {
		errorModal: ErrorModalState;
		onClose: () => void;
	}

	let { errorModal, onClose }: Props = $props();
</script>

<div class="fixed inset-0 bg-black/85 backdrop-blur-md flex items-center justify-center p-4 z-50">
	<div class="bg-neutral-900 border border-rose-900/60 rounded-2xl max-w-lg w-full p-6 space-y-4 shadow-2xl animate-in fade-in zoom-in duration-200">
		<div class="flex items-start gap-4">
			<div class="w-10 h-10 rounded-full bg-rose-500/10 border border-rose-500/30 flex items-center justify-center text-rose-400 shrink-0">
				<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
			</div>
			<div class="space-y-1">
				<h3 class="font-bold text-rose-300 text-base">{errorModal.title}</h3>
				<p class="text-xs text-neutral-300 leading-relaxed">{errorModal.message}</p>
			</div>
		</div>

		{#if errorModal.details}
			<div class="bg-neutral-950 p-3 rounded-xl border border-neutral-800 text-xs font-mono text-rose-400/90 overflow-x-auto max-h-36">
				{errorModal.details}
			</div>
		{/if}

		<div class="flex items-center justify-end pt-2">
			<button
				onclick={onClose}
				class="px-5 py-2 rounded-xl bg-neutral-800 hover:bg-neutral-700 text-neutral-100 text-xs font-semibold border border-neutral-700/60 transition-all"
			>
				Cancel & Dismiss
			</button>
		</div>
	</div>
</div>
