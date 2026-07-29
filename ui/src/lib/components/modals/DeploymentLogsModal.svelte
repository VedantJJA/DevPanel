<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	
	let { projectId, onClose } = $props<{ projectId: string, onClose: () => void }>();
	let logs = $state<string[]>([]);
	let eventSource: EventSource | null = null;
	let scrollContainer: HTMLElement | null = null;
	
	onMount(() => {
		// Subscribe to SSE deployment logs
		eventSource = new EventSource(`/api/projects/${projectId}/logs`);
		
		eventSource.onmessage = (event) => {
			try {
				const data = JSON.parse(event.data);
				const formattedMsg = `[${data.timestamp}] [${data.stage}] [${data.service}] ${data.message}`;
				logs = [...logs, formattedMsg];
				
				// Auto-scroll to bottom
				setTimeout(() => {
					if (scrollContainer) {
						scrollContainer.scrollTop = scrollContainer.scrollHeight;
					}
				}, 10);
			} catch (e) {
				console.error("Failed to parse log event", e);
			}
		};
		
		eventSource.onerror = () => {
			logs = [...logs, `[SYS] Log stream connection closed or failed.`];
			if (eventSource) {
				eventSource.close();
			}
		};
	});
	
	onDestroy(() => {
		if (eventSource) {
			eventSource.close();
		}
	});
</script>

<div class="fixed inset-0 z-50 overflow-y-auto" aria-labelledby="modal-title" role="dialog" aria-modal="true">
	<div class="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
		<div class="fixed inset-0 bg-gray-900/90 transition-opacity backdrop-blur-sm" aria-hidden="true" onclick={onClose}></div>
		<span class="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">&#8203;</span>
		
		<div class="inline-block align-bottom bg-[#09090b] rounded-xl text-left overflow-hidden shadow-2xl transform transition-all sm:my-8 sm:align-middle w-full max-w-4xl border border-gray-800">
			<div class="flex items-center justify-between px-4 py-3 bg-[#18181b] border-b border-gray-800">
				<div class="flex items-center gap-3">
					<div class="w-3 h-3 rounded-full bg-emerald-500 animate-pulse"></div>
					<h3 class="text-sm font-medium text-gray-300 font-mono">Deployment Logs: {projectId}</h3>
				</div>
				<button type="button" onclick={onClose} class="text-gray-500 hover:text-white transition-colors">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
				</button>
			</div>
			
			<div bind:this={scrollContainer} class="p-4 h-[60vh] overflow-y-auto font-mono text-xs text-gray-300 bg-black">
				{#if logs.length === 0}
					<div class="text-gray-600 italic">Waiting for deployment stream to begin...</div>
				{:else}
					{#each logs as log}
						<div class="whitespace-pre-wrap mb-1 {log.includes('[ERROR]') || log.toLowerCase().includes('error') ? 'text-red-400' : ''} {log.includes('[SUCCESS]') || log.includes('successfully') ? 'text-emerald-400' : ''} {log.includes('[SYS]') ? 'text-blue-400' : ''}">{log}</div>
					{/each}
				{/if}
			</div>
		</div>
	</div>
</div>
