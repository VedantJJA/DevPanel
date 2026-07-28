<script lang="ts">
	import type { LogStreamState } from '$lib/types';

	interface Props {
		selectedContainerLogs: LogStreamState;
		onClose: () => void;
	}

	let { selectedContainerLogs, onClose }: Props = $props();
</script>

<div class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50">
	<div class="bg-neutral-900 border border-neutral-800 rounded-2xl max-w-3xl w-full p-6 space-y-4 shadow-2xl">
		<div class="flex items-center justify-between border-b border-neutral-800 pb-3">
			<h3 class="font-bold text-neutral-100 flex items-center gap-2">
				<span class="text-emerald-400 font-mono">>_</span> Live WebSocket Logs: {selectedContainerLogs.name}
			</h3>
			<button onclick={onClose} class="text-neutral-400 hover:text-neutral-200 text-sm font-bold">✕</button>
		</div>
		<div class="bg-neutral-950 p-4 rounded-xl font-mono text-xs text-neutral-300 space-y-1.5 max-h-96 overflow-y-auto border border-neutral-800">
			{#each selectedContainerLogs.logs as logLine}
				<div class="hover:bg-neutral-900/50 px-1 py-0.5 rounded">{logLine}</div>
			{/each}
		</div>
	</div>
</div>
