<script lang="ts">
	import type { DeleteTarget } from '$lib/types';

	interface Props {
		deleteTarget: DeleteTarget;
		forceDelete: boolean;
		onForceChange: (val: boolean) => void;
		onConfirm: () => void;
		onCancel: () => void;
	}

	let { deleteTarget, forceDelete, onForceChange, onConfirm, onCancel }: Props = $props();
</script>

<div class="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 z-50">
	<div class="bg-neutral-900 border border-neutral-800 rounded-2xl max-w-md w-full p-6 space-y-5 shadow-2xl">
		<div class="flex items-center gap-3">
			<div class="w-10 h-10 rounded-full bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-400">
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
			</div>
			<div>
				<h3 class="font-bold text-neutral-100">Confirm Deletion</h3>
				<p class="text-xs text-neutral-400">Action cannot be undone</p>
			</div>
		</div>

		<p class="text-sm text-neutral-300">
			Are you sure you want to permanently delete {deleteTarget.type} <strong class="text-neutral-100 font-mono">{deleteTarget.label}</strong>?
		</p>

		<div class="flex items-center gap-2 pt-1">
			<input
				type="checkbox"
				id="forceCheck"
				checked={forceDelete}
				onchange={(e) => onForceChange(e.currentTarget.checked)}
				class="rounded bg-neutral-800 border-neutral-700 text-emerald-500 focus:ring-emerald-500/30"
			/>
			<label for="forceCheck" class="text-xs text-neutral-400 select-none cursor-pointer">Force deletion (force kill if running / in-use)</label>
		</div>

		<div class="flex items-center justify-end gap-3 pt-3 border-t border-neutral-800">
			<button
				onclick={onCancel}
				class="px-4 py-2 rounded-xl bg-neutral-800 hover:bg-neutral-700 text-neutral-300 text-xs font-semibold transition-all"
			>
				Cancel
			</button>
			<button
				onclick={onConfirm}
				class="px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold transition-all shadow-sm shadow-rose-950"
			>
				Confirm Delete
			</button>
		</div>
	</div>
</div>
