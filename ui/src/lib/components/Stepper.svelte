<script lang="ts">
	interface Props {
		currentStep: 1 | 2 | 3;
	}

	let { currentStep }: Props = $props();

	const steps = [
		{ number: 1, label: 'Select Repository', desc: 'Scan devpanel.yaml' },
		{ number: 2, label: 'Configure Services', desc: 'Environment & Ports' },
		{ number: 3, label: 'Deploy Terminal', desc: 'Live SSE Build Stream' }
	];
</script>

<div class="w-full max-w-3xl mx-auto mb-8">
	<div class="flex items-center justify-between relative">
		<!-- Connecting Line Background -->
		<div class="absolute left-0 top-1/2 -translate-y-1/2 w-full h-0.5 bg-neutral-800 -z-0"></div>
		<div
			class="absolute left-0 top-1/2 -translate-y-1/2 h-0.5 bg-emerald-500 transition-all duration-500 -z-0"
			style="width: {currentStep === 1 ? '0%' : currentStep === 2 ? '50%' : '100%'}"
		></div>

		{#each steps as step}
			<div class="flex flex-col items-center relative z-10">
				<div
					class="w-10 h-10 rounded-full font-bold text-xs flex items-center justify-center transition-all border-2 {step.number < currentStep
						? 'bg-emerald-600 border-emerald-500 text-white'
						: step.number === currentStep
						? 'bg-neutral-950 border-emerald-400 text-emerald-400 shadow-lg shadow-emerald-950'
						: 'bg-neutral-900 border-neutral-800 text-neutral-500'}"
				>
					{#if step.number < currentStep}
						✓
					{:else}
						{step.number}
					{/if}
				</div>

				<div class="mt-2 text-center">
					<div class="text-xs font-bold font-mono {step.number === currentStep ? 'text-emerald-400' : 'text-neutral-300'}">
						{step.label}
					</div>
					<div class="text-[10px] text-neutral-500">{step.desc}</div>
				</div>
			</div>
		{/each}
	</div>
</div>
