<script lang="ts">
	interface Props {
		envVars: Record<string, string>;
		onChange?: (updated: Record<string, string>) => void;
		readOnly?: boolean;
		serviceName?: string;
	}

	let { envVars = $bindable({}), onChange, readOnly = false, serviceName = 'service' }: Props = $props();

	let newKey = $state('');
	let newValue = $state('');
	let isBulkMode = $state(false);
	let bulkText = $state('');
	let visibleKeys = $state<Record<string, boolean>>({});
	let copiedKey = $state<string | null>(null);

	function toggleVisibility(key: string) {
		visibleKeys = { ...visibleKeys, [key]: !visibleKeys[key] };
	}

	function handleCopy(key: string, val: string) {
		if (navigator.clipboard) {
			navigator.clipboard.writeText(val);
			copiedKey = key;
			setTimeout(() => { if (copiedKey === key) copiedKey = null; }, 2000);
		}
	}

	function handleAdd() {
		if (!newKey.trim()) return;
		const k = newKey.trim().toUpperCase().replace(/[^A-Z0-9_]/g, '_');
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

	function openBulkEdit() {
		bulkText = Object.entries(envVars)
			.map(([k, v]) => `${k}=${v}`)
			.join('\n');
		isBulkMode = true;
	}

	function saveBulkEdit() {
		const updated: Record<string, string> = {};
		const lines = bulkText.split('\n');
		for (const line of lines) {
			const trimmed = line.trim();
			if (!trimmed || trimmed.startsWith('#')) continue;
			const eqIdx = trimmed.indexOf('=');
			if (eqIdx > 0) {
				const k = trimmed.slice(0, eqIdx).trim().toUpperCase().replace(/[^A-Z0-9_]/g, '_');
				const v = trimmed.slice(eqIdx + 1).trim();
				updated[k] = v;
			}
		}
		envVars = updated;
		isBulkMode = false;
		if (onChange) onChange(updated);
	}
</script>

<div class="space-y-6">
	<!-- Header Section matching stitch_modern_cloud_platform_dashboard(1) -->
	<div class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<h3 class="text-xl font-bold" style="color: var(--on-surface)">Environment Variables</h3>
			<p class="mt-1 text-xs" style="color: var(--on-surface-variant)">
				Configure keys and secrets for <span class="font-mono font-bold" style="color: var(--primary)">{serviceName}</span>. Changes are injected on the next deployment.
			</p>
		</div>

		{#if !readOnly}
			<button
				type="button"
				onclick={openBulkEdit}
				class="flex items-center gap-2 rounded-lg border px-4 py-2 text-xs font-semibold shadow-sm transition-colors hover:opacity-90"
				style="border-color: var(--outline-variant); background-color: var(--surface-lowest); color: var(--on-surface)"
			>
				<span class="material-symbols-outlined" style="font-size: 18px">edit_note</span>
				Bulk Edit (.env)
			</button>
		{/if}
	</div>

	{#if isBulkMode}
		<!-- Bulk Edit Mode Modal / Textarea -->
		<div class="card-surface p-6 space-y-4">
			<div class="flex items-center justify-between border-b pb-3" style="border-color: var(--outline-variant)">
				<h4 class="font-bold text-sm">Bulk Edit Raw Environment Variables (.env)</h4>
				<span class="text-xs" style="color: var(--on-surface-variant)">Format: KEY=VALUE (one per line)</span>
			</div>

			<textarea
				bind:value={bulkText}
				rows="10"
				placeholder="DATABASE_URL=postgres://admin:pass@host:5432/db&#10;API_KEY=sk_live_12345"
				class="w-full rounded-lg border p-4 font-mono text-xs outline-none transition-all"
				style="border-color: var(--outline-variant); background-color: var(--surface-low); color: var(--on-surface);"
			></textarea>

			<div class="flex justify-end gap-3">
				<button
					type="button"
					onclick={() => (isBulkMode = false)}
					class="rounded-lg px-4 py-2 text-xs font-medium hover:opacity-80"
					style="color: var(--on-surface-variant)"
				>
					Cancel
				</button>
				<button
					type="button"
					onclick={saveBulkEdit}
					class="flex items-center gap-2 rounded-lg px-5 py-2 text-xs font-semibold shadow-sm"
					style="background-color: var(--primary); color: var(--on-primary)"
				>
					<span class="material-symbols-outlined" style="font-size: 16px">check</span>
					Apply Bulk Changes
				</button>
			</div>
		</div>

	{:else}

		<!-- Variable List Table Card -->
		<div class="card-surface overflow-hidden rounded-xl border shadow-sm" style="border-color: var(--outline-variant)">
			<!-- Table Header -->
			<div class="grid grid-cols-[1.2fr_1.8fr_100px] gap-4 border-b px-6 py-3 text-[11px] font-bold uppercase tracking-widest" style="border-color: var(--outline-variant); background-color: var(--surface-low); color: var(--outline)">
				<div>Variable Key</div>
				<div>Value</div>
				<div class="text-right">Actions</div>
			</div>

			<!-- Table Body -->
			<div class="divide-y" style="border-color: var(--outline-variant)">
				{#if Object.keys(envVars).length === 0}
					<div class="p-8 text-center text-xs italic" style="color: var(--on-surface-variant)">
						No environment variables set. Use the row below or Bulk Edit to add variables.
					</div>
				{:else}
					{#each Object.entries(envVars) as [k, v] (k)}
						<div class="grid grid-cols-[1.2fr_1.8fr_100px] items-center gap-4 px-6 py-4 transition-colors hover:bg-[color:var(--surface-low)] group">
							<div class="font-mono text-sm font-semibold truncate" style="color: var(--on-surface)">{k}</div>

							<!-- Value Input with Secret Toggle -->
							<div class="relative flex items-center">
								<input
									type={visibleKeys[k] ? 'text' : 'password'}
									bind:value={envVars[k]}
									disabled={readOnly}
									oninput={() => onChange && onChange(envVars)}
									class="w-full border-none bg-transparent p-0 font-mono text-sm outline-none focus:ring-0 disabled:opacity-60"
									style="color: var(--on-surface-variant)"
								/>
								<button
									type="button"
									onclick={() => toggleVisibility(k)}
									class="absolute right-0 text-xs transition-colors hover:opacity-80"
									style="color: var(--outline)"
									title={visibleKeys[k] ? 'Hide value' : 'Show value'}
								>
									<span class="material-symbols-outlined" style="font-size: 18px">
										{visibleKeys[k] ? 'visibility_off' : 'visibility'}
									</span>
								</button>
							</div>

							<!-- Actions -->
							<div class="flex items-center justify-end gap-2 opacity-80 group-hover:opacity-100">
								<button
									type="button"
									onclick={() => handleCopy(k, envVars[k])}
									class="rounded p-1.5 transition-colors hover:bg-[color:var(--surface-high)]"
									style="color: var(--outline)"
									title="Copy value"
								>
									<span class="material-symbols-outlined" style="font-size: 18px">
										{copiedKey === k ? 'check' : 'content_copy'}
									</span>
								</button>
								{#if !readOnly}
									<button
										type="button"
										onclick={() => handleRemove(k)}
										class="rounded p-1.5 transition-colors hover:bg-[color:var(--error-container)]"
										style="color: var(--error)"
										title="Delete variable"
									>
										<span class="material-symbols-outlined" style="font-size: 18px">delete</span>
									</button>
								{/if}
							</div>
						</div>
					{/each}
				{/if}

				<!-- Add Variable Row -->
				{#if !readOnly}
					<div class="p-4" style="background-color: var(--surface-low)">
						<div class="grid grid-cols-[1.2fr_1.8fr_100px] items-center gap-3 rounded-lg border-2 border-dashed p-2 transition-all hover:border-[color:var(--primary)]" style="border-color: var(--outline-variant)">
							<input
								type="text"
								bind:value={newKey}
								placeholder="KEY_NAME"
								class="bg-transparent border-none font-mono text-sm font-bold uppercase outline-none focus:ring-0"
								style="color: var(--on-surface)"
							/>
							<input
								type="text"
								bind:value={newValue}
								placeholder="value"
								class="bg-transparent border-none font-mono text-sm outline-none focus:ring-0"
								style="color: var(--on-surface)"
							/>
							<button
								type="button"
								onclick={handleAdd}
								class="rounded-lg py-2 text-center text-xs font-bold uppercase tracking-wider shadow-sm transition-all hover:opacity-90"
								style="background-color: var(--primary); color: var(--on-primary)"
							>
								Add
							</button>
						</div>
					</div>
				{/if}
			</div>
		</div>

		<!-- Secrets Integration Alert Banner from stitch_modern_cloud_platform_dashboard(1) -->
		<div class="flex items-start gap-4 rounded-xl border p-5 shadow-sm" style="border-color: var(--outline-variant); background-color: var(--surface-lowest)">
			<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg" style="background-color: var(--primary-fixed); color: var(--primary)">
				<span class="material-symbols-outlined" style="font-size: 22px">lock</span>
			</div>
			<div class="flex-1">
				<h4 class="font-bold text-sm" style="color: var(--on-surface)">External Secrets Integration</h4>
				<p class="mt-1 text-xs leading-relaxed" style="color: var(--on-surface-variant)">
					Need to sync production secrets automatically? You can connect external secret providers like <span class="font-bold">HashiCorp Vault</span> or <span class="font-bold">AWS Secrets Manager</span>.
				</p>
			</div>
		</div>
	{/if}
</div>
