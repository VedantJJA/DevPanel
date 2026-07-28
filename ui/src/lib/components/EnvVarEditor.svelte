<script lang="ts">
	interface Props {
		envVars: Record<string, string>;
		onChange?: (updated: Record<string, string>) => void;
		readOnly?: boolean;
	}

	let { envVars = $bindable({}), onChange, readOnly = false }: Props = $props();

	let newKey = $state('');
	let newValue = $state('');

	function handleAdd() {
		if (!newKey.trim()) return;
		const k = newKey.trim().toUpperCase();
		const updated = { ...envVars, [k]: newValue.trim() };
		envVars = updated;
		newKey = '';
		newValue = '';
		if (onChange) onChange(updated);
	}

	function handleRemove(key: string) {
		const updated = { ...envVars };
		delete updated[key];
		envVars = updated;
		if (onChange) onChange(updated);
	}
</script>

<div class="space-y-3 font-mono text-xs">
	{#if Object.keys(envVars).length === 0}
		<div class="p-3 rounded-xl bg-neutral-950/60 border border-neutral-800 text-neutral-500 italic text-center">
			No environment variables configured.
		</div>
	{:else}
		<div class="space-y-2 max-h-56 overflow-y-auto pr-1">
			{#each Object.entries(envVars) as [k, v] (k)}
				<div class="flex items-center gap-2">
					<input
						type="text"
						value={k}
						readonly
						class="w-1/3 px-3 py-1.5 bg-neutral-950 border border-neutral-800 rounded-lg text-neutral-300 font-semibold"
					/>
					<input
						type="text"
						bind:value={envVars[k]}
						disabled={readOnly}
						oninput={() => onChange && onChange(envVars)}
						class="flex-1 px-3 py-1.5 bg-neutral-950 border border-neutral-800 rounded-lg text-emerald-400 focus:outline-none focus:border-emerald-500 disabled:opacity-60"
					/>
					{#if !readOnly}
						<button
							type="button"
							onclick={() => handleRemove(k)}
							class="p-1.5 rounded-lg bg-neutral-800 hover:bg-rose-600/30 text-rose-400 transition-all font-bold"
							title="Remove variable"
						>
							✕
						</button>
					{/if}
				</div>
			{/each}
		</div>
	{/if}

	{#if !readOnly}
		<div class="pt-3 border-t border-neutral-800 flex items-center gap-2">
			<input
				type="text"
				bind:value={newKey}
				placeholder="KEY_NAME"
				class="w-1/3 px-3 py-1.5 bg-neutral-950 border border-neutral-800 rounded-lg text-neutral-100 uppercase focus:outline-none focus:border-emerald-500"
			/>
			<input
				type="text"
				bind:value={newValue}
				placeholder="value"
				class="flex-1 px-3 py-1.5 bg-neutral-950 border border-neutral-800 rounded-lg text-neutral-100 focus:outline-none focus:border-emerald-500"
			/>
			<button
				type="button"
				onclick={handleAdd}
				class="px-3 py-1.5 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-emerald-400 font-semibold text-xs transition-all"
			>
				+ Add
			</button>
		</div>
	{/if}
</div>
