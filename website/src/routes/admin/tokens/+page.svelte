<script>
	import { onMount } from 'svelte';

	let tokens = [];
	let loading = true;
	let error = '';
	let success = '';
	let newName = '';
	let createdToken = '';

	onMount(() => {
		loadTokens();
	});

	async function loadTokens() {
		const token = localStorage.getItem('auth_token');
		loading = true;
		try {
			const res = await fetch('/api/admin/tokens', {
				headers: { Authorization: `Bearer ${token}` }
			});
			if (res.ok) {
				const data = await res.json();
				tokens = data.tokens || [];
			} else {
				error = '加载 Token 列表失败';
			}
		} catch (e) {
			error = '连接服务器失败';
		} finally {
			loading = false;
		}
	}

	async function handleCreate() {
		const token = localStorage.getItem('auth_token');
		error = '';
		success = '';
		createdToken = '';
		try {
			const res = await fetch('/api/admin/tokens', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify({ name: newName.trim() || '未命名 Token' })
			});
			if (res.ok) {
				const data = await res.json();
				createdToken = data.token;
				success = `Token "${data.name}" 已创建，请立即复制保存，关闭后将无法再次查看完整 Token。`;
				newName = '';
				loadTokens();
			} else {
				error = '创建 Token 失败';
			}
		} catch (e) {
			error = '连接服务器失败';
		}
	}

	async function handleDelete(id, name) {
		if (!confirm(`确认删除 Token "${name}"？此操作不可撤销。`)) return;
		const token = localStorage.getItem('auth_token');
		error = '';
		success = '';
		try {
			const res = await fetch(`/api/admin/tokens/${id}`, {
				method: 'DELETE',
				headers: { Authorization: `Bearer ${token}` }
			});
			if (res.ok) {
				success = 'Token 已删除';
				loadTokens();
			} else {
				error = '删除 Token 失败';
			}
		} catch (e) {
			error = '连接服务器失败';
		}
	}

	function copyToClipboard(text) {
		navigator.clipboard.writeText(text).then(() => {
			success = '已复制到剪贴板';
		});
	}
</script>

<div class="max-w-3xl">
	<h1 class="text-2xl font-bold mb-6">API Token 管理</h1>

	{#if error}
		<div class="alert alert-error mb-4">
			<span>{error}</span>
		</div>
	{/if}
	{#if success}
		<div class="alert alert-success mb-4">
			<span>{success}</span>
		</div>
	{/if}

	{#if createdToken}
		<div class="alert alert-warning mb-4 flex-col items-start gap-2">
			<span class="font-bold">请立即复制保存此 Token（仅显示一次）：</span>
			<div class="flex items-center gap-2 w-full">
				<code class="text-sm bg-base-300 px-2 py-1 rounded flex-1 break-all">{createdToken}</code>
				<button class="btn btn-sm btn-outline" on:click={() => copyToClipboard(createdToken)}>
					复制
				</button>
			</div>
		</div>
	{/if}

	<!-- 创建新 Token -->
	<div class="card bg-base-200 mb-6">
		<div class="card-body">
			<h2 class="card-title text-lg">生成新 Token</h2>
			<form on:submit|preventDefault={handleCreate} class="flex gap-2">
				<input
					bind:value={newName}
					type="text"
					placeholder="Token 名称（可选）"
					class="input input-bordered flex-1"
				/>
				<button type="submit" class="btn btn-primary">生成</button>
			</form>
		</div>
	</div>

	<!-- Token 列表 -->
	{#if loading}
		<div class="flex justify-center py-8">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if tokens.length === 0}
		<div class="text-center py-8 text-base-content/50">暂无 API Token</div>
	{:else}
		<div class="overflow-x-auto">
			<table class="table table-zebra">
				<thead>
					<tr>
						<th>名称</th>
						<th>Token</th>
						<th>创建时间</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					{#each tokens as t}
						<tr>
							<td>{t.name || '未命名'}</td>
							<td>
								<code class="text-xs">{t.token}</code>
							</td>
							<td class="text-sm">{t.created_at}</td>
							<td>
								<button
									class="btn btn-sm btn-error btn-outline"
									on:click={() => handleDelete(t.id, t.name)}
								>
									删除
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
