<script>
	import { onMount } from 'svelte';

	let results = [];
	let labels = {};
	let loading = true;
	let error = '';
	let success = '';

	onMount(loadResults);

	async function loadResults() {
		const token = localStorage.getItem('auth_token');
		loading = true;
		try {
			const res = await fetch('/api/admin/ai/results', {
				headers: { Authorization: `Bearer ${token}` }
			});
			if (!res.ok) throw new Error();
			const data = await res.json();
			results = data.results || [];
			labels = data.labels || {};
		} catch (e) {
			error = '加载 AI 结果失败';
		} finally {
			loading = false;
		}
	}

	async function labelResult(text, kind) {
		const token = localStorage.getItem('auth_token');
		error = '';
		success = '';
		try {
			const res = await fetch('/api/admin/ai/results/label', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({ text, kind })
			});
			if (!res.ok) throw new Error();
			labels = { ...labels, [text]: kind };
			success = `已标记为 ${kind}`;
		} catch (e) {
			error = '标记词条失败';
		}
	}
</script>

<div class="max-w-4xl">
	<h1 class="text-2xl font-bold mb-2">AI 最近结果</h1>
	<p class="text-sm text-base-content/60 mb-6">显示最近 100 条已接受的 AI 词条。标记会保存到长期词条库。</p>

	{#if error}
		<div class="alert alert-error mb-4"><span>{error}</span></div>
	{/if}
	{#if success}
		<div class="alert alert-success mb-4"><span>{success}</span></div>
	{/if}

	{#if loading}
		<div class="flex justify-center py-8"><span class="loading loading-spinner loading-lg"></span></div>
	{:else if results.length === 0}
		<div class="text-center py-8 text-base-content/50">暂无 AI 生成结果</div>
	{:else}
		<div class="overflow-x-auto">
			<table class="table table-zebra">
				<thead>
					<tr>
						<th>词条</th>
						<th>适用小时</th>
						<th>生成时间</th>
						<th>标签</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					{#each results as result}
						<tr>
							<td class="font-medium">{result.text}</td>
							<td class="text-sm whitespace-nowrap">{result.hour}</td>
							<td class="text-sm whitespace-nowrap">{result.generated_at}</td>
							<td>
								{#if labels[result.text] === 'good'}
									<span class="badge badge-success">good</span>
								{:else if labels[result.text] === 'bad'}
									<span class="badge badge-error">bad</span>
								{:else}
									<span class="text-base-content/50">未标记</span>
								{/if}
							</td>
							<td class="flex gap-2">
								<button
									class="btn btn-success btn-outline btn-xs"
									on:click={() => labelResult(result.text, 'good')}>good</button
								>
								<button
									class="btn btn-error btn-outline btn-xs"
									on:click={() => labelResult(result.text, 'bad')}>bad</button
								>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
