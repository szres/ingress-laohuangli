<script>
	import { onMount } from 'svelte';

	let good = [];
	let bad = [];
	let loading = true;
	let error = '';
	let success = '';

	onMount(loadEntries);

	async function loadEntries() {
		const token = localStorage.getItem('auth_token');
		loading = true;
		try {
			const headers = { Authorization: `Bearer ${token}` };
			const [goodRes, badRes] = await Promise.all([
				fetch('/api/admin/ai/curated/good', { headers }),
				fetch('/api/admin/ai/curated/bad', { headers })
			]);
			if (!goodRes.ok || !badRes.ok) throw new Error();
			good = (await goodRes.json()).entries || [];
			bad = (await badRes.json()).entries || [];
		} catch (e) {
			error = '加载长期词条库失败';
		} finally {
			loading = false;
		}
	}

	async function deleteEntry(text, kind) {
		if (!confirm(`确认从 ${kind} 池删除“${text}”？`)) return;
		const token = localStorage.getItem('auth_token');
		error = '';
		success = '';
		try {
			const res = await fetch(`/api/admin/ai/curated/${kind}`, {
				method: 'DELETE',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({ text })
			});
			if (!res.ok) throw new Error();
			if (kind === 'good') good = good.filter((entry) => entry !== text);
			else bad = bad.filter((entry) => entry !== text);
			success = '词条已删除';
		} catch (e) {
			error = '删除词条失败';
		}
	}
</script>

<div class="max-w-4xl">
	<h1 class="text-2xl font-bold mb-2">AI 长期词条库</h1>
	<p class="text-sm text-base-content/60 mb-6">good 用作优质风格参考；bad 用作负面示例，提示模型避免类似内容。</p>

	{#if error}
		<div class="alert alert-error mb-4"><span>{error}</span></div>
	{/if}
	{#if success}
		<div class="alert alert-success mb-4"><span>{success}</span></div>
	{/if}

	{#if loading}
		<div class="flex justify-center py-8"><span class="loading loading-spinner loading-lg"></span></div>
	{:else}
		<div class="grid gap-6 lg:grid-cols-2">
			<section class="card bg-base-200">
				<div class="card-body">
					<h2 class="card-title">good <span class="badge badge-success">{good.length}</span></h2>
					{#if good.length === 0}
						<p class="text-base-content/50">暂无词条</p>
					{:else}
						<ul class="flex flex-col gap-2">
							{#each good as entry}
								<li class="flex items-center justify-between gap-3">
									<span class="break-all">{entry}</span>
									<button
										class="btn btn-error btn-outline btn-xs shrink-0"
										on:click={() => deleteEntry(entry, 'good')}>删除</button
									>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</section>
			<section class="card bg-base-200">
				<div class="card-body">
					<h2 class="card-title">bad <span class="badge badge-error">{bad.length}</span></h2>
					{#if bad.length === 0}
						<p class="text-base-content/50">暂无词条</p>
					{:else}
						<ul class="flex flex-col gap-2">
							{#each bad as entry}
								<li class="flex items-center justify-between gap-3">
									<span class="break-all">{entry}</span>
									<button
										class="btn btn-error btn-outline btn-xs shrink-0"
										on:click={() => deleteEntry(entry, 'bad')}>删除</button
									>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</section>
		</div>
	{/if}
</div>
