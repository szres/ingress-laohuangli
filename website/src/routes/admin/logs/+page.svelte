<script>
	export let data;
	import { onMount, onDestroy } from 'svelte';
	import { invalidateAll } from '$app/navigation';

	let autoRefresh = true;
	let interval;
	let logs = data.logs || [];
	let logContainer;

	$: logs = data.logs || [];

	onMount(() => {
		interval = setInterval(() => {
			if (autoRefresh) {
				invalidateAll();
			}
		}, 5000);
		// 滚动到底部
		if (logContainer) {
			logContainer.scrollTop = logContainer.scrollHeight;
		}
	});

	onDestroy(() => {
		if (interval) clearInterval(interval);
	});

	function scrollToBottom() {
		if (logContainer) {
			setTimeout(() => {
				logContainer.scrollTop = logContainer.scrollHeight;
			}, 100);
		}
	}

	$: if (logs.length > 0) {
		scrollToBottom();
	}
</script>

<div class="flex flex-col h-full">
	<div class="flex items-center justify-between mb-4">
		<h1 class="text-2xl font-bold">日志查看</h1>
		<div class="flex items-center gap-4">
			<label class="flex items-center gap-2 cursor-pointer">
				<input type="checkbox" bind:checked={autoRefresh} class="toggle toggle-primary toggle-sm" />
				<span class="text-sm">自动刷新 (5s)</span>
			</label>
			<button class="btn btn-sm btn-outline" on:click={() => invalidateAll()}>手动刷新</button>
			<span class="badge badge-neutral">{logs.length} 行</span>
		</div>
	</div>

	<div
		bind:this={logContainer}
		class="flex-1 bg-base-300 rounded-lg p-4 overflow-auto font-mono text-sm"
	>
		{#if logs.length === 0}
			<div class="text-center text-base-content/50 py-8">暂无日志</div>
		{:else}
			{#each logs as line, i}
				<div
					class="whitespace-pre-wrap border-b border-base-content/5 py-0.5 hover:bg-base-content/5"
				>
					<span class="text-base-content/40 select-none mr-2">{i + 1}</span>
					{line}
				</div>
			{/each}
		{/if}
	</div>
</div>
