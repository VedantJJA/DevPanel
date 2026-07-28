<script lang="ts">
	interface Props {
		autoRefreshRateSec: number;
		actionLoading: string | null;
		onSetAutoRefresh: (rate: number) => void;
		onPruneSystem: () => void;
	}

	let { autoRefreshRateSec, actionLoading, onSetAutoRefresh, onPruneSystem }: Props = $props();
</script>

<section class="space-y-6 max-w-4xl">
	<h3 class="text-lg font-bold text-neutral-100">DevPanel Server Settings</h3>

	<div class="space-y-4">
		<!-- Auto Refresh Rate Controls -->
		<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-3">
			<h4 class="font-semibold text-neutral-200 text-sm">Dashboard Auto-Refresh Interval</h4>
			<p class="text-xs text-neutral-400">Configure how frequently telemetry and Docker stats update.</p>
			<div class="flex items-center gap-2 pt-2">
				{#each [2, 5, 10] as rate}
					<button
						onclick={() => onSetAutoRefresh(rate)}
						class="px-3.5 py-1.5 rounded-lg text-xs font-mono font-semibold transition-all border {autoRefreshRateSec === rate ? 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30' : 'bg-neutral-800 text-neutral-400 border-neutral-700'}"
					>{rate}s</button>
				{/each}
			</div>
		</div>

		<!-- System Prune Action -->
		<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-3">
			<div class="flex items-center justify-between">
				<div>
					<h4 class="font-semibold text-neutral-200 text-sm">Prune Unused Docker System Resources</h4>
					<p class="text-xs text-neutral-400 mt-1">Remove all stopped containers and dangling volumes to free disk space.</p>
				</div>
				<button
					onclick={onPruneSystem}
					disabled={actionLoading === 'prune'}
					class="px-4 py-2 rounded-xl bg-rose-600/20 text-rose-400 hover:bg-rose-600 hover:text-white border border-rose-500/30 text-xs font-semibold transition-all disabled:opacity-50"
				>
					{actionLoading === 'prune' ? 'Pruning...' : 'Prune Unused'}
				</button>
			</div>
		</div>

		<!-- Socket Activation Settings -->
		<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-3">
			<div class="flex items-center justify-between">
				<h4 class="font-semibold text-neutral-200 text-sm">Systemd Socket Activation</h4>
				<span class="px-2.5 py-1 rounded-full text-xs font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">Enabled</span>
			</div>
			<p class="text-xs text-neutral-400">Listens on port 80/443 via systemd sockets (`LISTEN_FDS=1`). Server scales to zero after 5 minutes of idle requests.</p>
		</div>

		<!-- Caddy Reverse Proxy Config -->
		<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-3">
			<h4 class="font-semibold text-neutral-200 text-sm">Caddy Admin API Integration</h4>
			<p class="text-xs text-neutral-400">Dynamic reverse-proxy route injection and On-Demand TLS handshake verification.</p>
			<div class="pt-2 text-xs font-mono text-neutral-300">
				<span class="text-neutral-500">Admin Endpoint:</span> http://localhost:2019
			</div>
		</div>

		<!-- SQLite Database -->
		<div class="p-6 rounded-2xl border border-neutral-800 bg-neutral-900/60 space-y-3">
			<h4 class="font-semibold text-neutral-200 text-sm">Pure-Go SQLite Storage</h4>
			<p class="text-xs text-neutral-400">WAL mode enabled for concurrent read/write operations without CGo dependencies.</p>
			<div class="pt-2 text-xs font-mono text-neutral-300">
				<span class="text-neutral-500">Database File:</span> ./devpnl.db
			</div>
		</div>
	</div>
</section>
