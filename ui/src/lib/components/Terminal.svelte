<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { LogEvent } from '../types';

	interface Props {
		projectId: string;
		serviceFilter?: string;
		title?: string;
		sourceUrl?: string;
	}

	let { projectId, serviceFilter = '', title = 'Live Deployment & Build Logs', sourceUrl = '' }: Props = $props();

	let logs = $state<LogEvent[]>([]);
	let sseConnected = $state(false);
	let terminalContainer: HTMLDivElement | null = $state(null);
	let eventSource: EventSource | null = null;

	function scrollToBottom() {
		if (terminalContainer) {
			terminalContainer.scrollTop = terminalContainer.scrollHeight;
		}
	}

	$effect(() => {
		if (eventSource) eventSource.close();
		logs = [];
		sseConnected = false;

		let sseUrl = sourceUrl;
		if (!sseUrl) {
			sseUrl = `/api/projects/${encodeURIComponent(projectId)}/logs`;
			if (serviceFilter) {
				sseUrl += `?service=${encodeURIComponent(serviceFilter)}`;
			}
		}

		eventSource = new EventSource(sseUrl);

		eventSource.onopen = () => {
			sseConnected = true;
		};

		eventSource.addEventListener('connected', () => {
			sseConnected = true;
		});

		eventSource.onmessage = (event) => {
			try {
				const evt: LogEvent = JSON.parse(event.data);
				logs = [...logs, evt];
				setTimeout(scrollToBottom, 40);
			} catch (e) {
				logs = [
					...logs,
					{
						timestamp: new Date().toISOString(),
						stage: 'system',
						service: 'console',
						message: event.data,
						level: 'info'
					}
				];
				setTimeout(scrollToBottom, 40);
			}
		};

		eventSource.onerror = () => {
			sseConnected = false;
		};

		return () => {
			if (eventSource) eventSource.close();
		};
	});
</script>

<div class="flex flex-col h-full bg-black rounded-2xl border border-neutral-800 shadow-2xl overflow-hidden font-mono text-xs">
	<!-- Terminal Bar -->
	<div class="px-4 py-3 bg-neutral-900 border-b border-neutral-800 flex items-center justify-between shrink-0">
		<div class="flex items-center gap-2">
			<span class="w-3 h-3 rounded-full bg-rose-500/80"></span>
			<span class="w-3 h-3 rounded-full bg-amber-500/80"></span>
			<span class="w-3 h-3 rounded-full bg-emerald-500/80"></span>
			<span class="ml-2 text-neutral-400 text-xs font-semibold">{title}</span>
		</div>

		<div class="flex items-center gap-3">
			<button onclick={() => (logs = [])} class="text-neutral-400 hover:text-neutral-200 text-xs">Clear</button>
			<span class="w-2 h-2 rounded-full {sseConnected ? 'bg-emerald-400 animate-pulse' : 'bg-rose-500'}"></span>
		</div>
	</div>

	<!-- Log Viewport -->
	<div bind:this={terminalContainer} class="flex-1 p-4 overflow-y-auto space-y-1.5 scroll-smooth">
		{#if logs.length === 0}
			<div class="text-neutral-500 italic py-4">Connecting to real-time build and deploy SSE stream...</div>
		{/if}

		{#each logs as item, i (i)}
			<div class="flex items-start gap-3 leading-relaxed">
				<span class="text-neutral-600 text-[11px] shrink-0">{item.timestamp.slice(11, 19)}</span>
				<span class="px-1.5 py-0.5 rounded text-[10px] uppercase font-bold shrink-0 {item.stage === 'build'
					? 'bg-sky-950 text-sky-400 border border-sky-800'
					: item.stage === 'deploy'
					? 'bg-purple-950 text-purple-400 border border-purple-800'
					: item.stage === 'clone'
					? 'bg-amber-950 text-amber-400 border border-amber-800'
					: 'bg-neutral-800 text-neutral-300'}">
					{item.stage}
				</span>
				<span class="text-neutral-400 shrink-0">[{item.service}]</span>
				<span class="flex-1 whitespace-pre-wrap {item.level === 'error'
					? 'text-rose-400 font-bold'
					: item.level === 'success'
					? 'text-emerald-400 font-bold'
					: item.level === 'warn'
					? 'text-amber-300'
					: 'text-neutral-200'}">
					{item.message}
				</span>
			</div>
		{/each}
	</div>
</div>
