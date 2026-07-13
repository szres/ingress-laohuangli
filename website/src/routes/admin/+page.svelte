<script>
	import { onMount } from 'svelte';

	let config = {};
	let loading = true;
	let success = '';
	let error = '';

	const fields = [
		'bot_token',
		'admin_id',
		'kuma_push_url',
		'openai_api_key',
		'openai_base_url',
		'openai_model',
		'admin_username',
		'admin_password',
		'web_domain'
	];

	let formData = {};

	onMount(async () => {
		const token = localStorage.getItem('auth_token');
		try {
			const res = await fetch('/api/admin/config', {
				headers: { Authorization: `Bearer ${token}` }
			});
			if (res.ok) {
				config = await res.json();
			}
		} catch (e) {
			error = '加载配置失败';
		} finally {
			loading = false;
		}
	});

	async function handleSave() {
		const token = localStorage.getItem('auth_token');
		const update = {};

		for (const field of fields) {
			const value = formData[field];
			if (value && value.trim() !== '') {
				update[field] = value.trim();
			}
		}

		if (Object.keys(update).length === 0) {
			error = '没有需要更新的配置';
			return;
		}

		try {
			const res = await fetch('/api/admin/config', {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify(update)
			});

			if (!res.ok) {
				error = '保存配置失败';
				return;
			}

			success = '配置已保存';
			error = '';
			// 重新加载配置
			const configRes = await fetch('/api/admin/config', {
				headers: { Authorization: `Bearer ${token}` }
			});
			if (configRes.ok) {
				config = await configRes.json();
			}
			// 清空表单
			formData = {};
		} catch (e) {
			error = '连接服务器失败';
		}
	}
</script>

<div class="max-w-2xl">
	<h1 class="text-2xl font-bold mb-6">配置管理</h1>

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

	{#if loading}
		<div class="flex justify-center py-8">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else}
		<div class="mb-4 p-4 bg-base-200 rounded-lg">
			<p class="text-sm text-base-content/70">
				💡 敏感字段（Token、API Key）已脱敏显示。留空表示不修改，填入新值则更新。
			</p>
		</div>

		<form on:submit|preventDefault={handleSave} class="flex flex-col gap-6">
			<!-- Telegram Bot 配置 -->
			<div class="card bg-base-200">
				<div class="card-body">
					<h2 class="card-title">🤖 Telegram Bot</h2>
					<div class="form-control">
						<label class="label" for="bot_token">
							<span class="label-text">BOT_TOKEN</span>
						</label>
						<input
							id="bot_token"
							bind:value={formData.bot_token}
							type="text"
							placeholder={config.bot_token || '留空不修改'}
							class="input input-bordered"
						/>
					</div>
					<div class="form-control">
						<label class="label" for="admin_id">
							<span class="label-text">BOT_ADMIN_ID</span>
							<span class="label-text-alt">当前: {config.admin_id || '未设置'}</span>
						</label>
						<input
							id="admin_id"
							bind:value={formData.admin_id}
							type="text"
							placeholder="留空不修改"
							class="input input-bordered"
						/>
					</div>
					<div class="form-control">
						<label class="label" for="web_domain">
							<span class="label-text">WEB_DOMAIN</span>
							<span class="label-text-alt">当前: {config.web_domain || '未设置'}</span>
						</label>
						<input
							id="web_domain"
							bind:value={formData.web_domain}
							type="text"
							placeholder="留空不修改"
							class="input input-bordered"
						/>
					</div>
					<div class="form-control">
						<label class="label" for="kuma_push_url">
							<span class="label-text">KUMA_PUSH_URL</span>
							<span class="label-text-alt">当前: {config.kuma_push_url || '未设置'}</span>
						</label>
						<input
							id="kuma_push_url"
							bind:value={formData.kuma_push_url}
							type="text"
							placeholder="留空不修改"
							class="input input-bordered"
						/>
					</div>
				</div>
			</div>

			<!-- AI 配置 -->
			<div class="card bg-base-200">
				<div class="card-body">
					<h2 class="card-title">🧠 AI 配置</h2>
					<p class="text-sm text-base-content/60 mb-2">修改后会自动热重载，无需重启</p>
					<div class="form-control">
						<label class="label" for="openai_api_key">
							<span class="label-text">OpenAI API Key</span>
						</label>
						<input
							id="openai_api_key"
							bind:value={formData.openai_api_key}
							type="text"
							placeholder={config.openai_api_key || '留空不修改'}
							class="input input-bordered"
						/>
					</div>
					<div class="form-control">
						<label class="label" for="openai_base_url">
							<span class="label-text">OpenAI Base URL</span>
							<span class="label-text-alt">当前: {config.openai_base_url || '默认'}</span>
						</label>
						<input
							id="openai_base_url"
							bind:value={formData.openai_base_url}
							type="text"
							placeholder="留空不修改"
							class="input input-bordered"
						/>
					</div>
					<div class="form-control">
						<label class="label" for="openai_model">
							<span class="label-text">OpenAI 模型（可多个）</span>
							<span class="label-text-alt"
								>当前: {(config.openai_model || 'gpt-4o-mini').replace(/[\n\r;]+/g, ', ')}</span
							>
						</label>
						<textarea
							id="openai_model"
							bind:value={formData.openai_model}
							placeholder={'每行一个模型，按顺序优先调用，失败才切换\n例如:\ngpt-4o-mini\ngpt-4o'}
							class="textarea textarea-bordered font-mono text-sm"
							rows="4"
						></textarea>
						<label class="label" for="openai_model">
							<span class="label-text-alt">留空不修改；首个模型优先，失败时按顺序 fallback，各自独立退避重试</span>
						</label>
					</div>
				</div>
			</div>

			<!-- 管理员账户 -->
			<div class="card bg-base-200">
				<div class="card-body">
					<h2 class="card-title">👤 管理员账户</h2>
					<div class="form-control">
						<label class="label" for="admin_username">
							<span class="label-text">用户名</span>
							<span class="label-text-alt">当前: {config.admin_username || '未设置'}</span>
						</label>
						<input
							id="admin_username"
							bind:value={formData.admin_username}
							type="text"
							placeholder="留空不修改"
							class="input input-bordered"
						/>
					</div>
					<div class="form-control">
						<label class="label" for="admin_password">
							<span class="label-text">密码</span>
						</label>
						<input
							id="admin_password"
							bind:value={formData.admin_password}
							type="password"
							placeholder="留空不修改"
							class="input input-bordered"
						/>
					</div>
				</div>
			</div>

			<button type="submit" class="btn btn-primary btn-lg">保存配置</button>
		</form>
	{/if}
</div>
