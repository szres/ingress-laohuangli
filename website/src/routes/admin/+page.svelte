<script>
	export let data;
	export let form;

	let config = data.config || {};
</script>

<div class="max-w-2xl">
	<h1 class="text-2xl font-bold mb-6">配置管理</h1>

	{#if form?.error}
		<div class="alert alert-error mb-4">
			<span>{form.error}</span>
		</div>
	{/if}
	{#if form?.success}
		<div class="alert alert-success mb-4">
			<span>{form.success}</span>
		</div>
	{/if}

	<div class="mb-4 p-4 bg-base-200 rounded-lg">
		<p class="text-sm text-base-content/70">
			💡 敏感字段（Token、API Key）已脱敏显示。留空表示不修改，填入新值则更新。
		</p>
	</div>

	<form method="POST" action="?/save" class="flex flex-col gap-6">
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
						name="bot_token"
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
						name="admin_id"
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
						name="kuma_push_url"
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
						name="openai_api_key"
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
						name="openai_base_url"
						type="text"
						placeholder="留空不修改"
						class="input input-bordered"
					/>
				</div>
				<div class="form-control">
					<label class="label" for="openai_model">
						<span class="label-text">OpenAI 模型</span>
						<span class="label-text-alt">当前: {config.openai_model || 'gpt-4o-mini'}</span>
					</label>
					<input
						id="openai_model"
						name="openai_model"
						type="text"
						placeholder="留空不修改"
						class="input input-bordered"
					/>
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
						name="admin_username"
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
						name="admin_password"
						type="password"
						placeholder="留空不修改"
						class="input input-bordered"
					/>
				</div>
			</div>
		</div>

		<button type="submit" class="btn btn-primary btn-lg">保存配置</button>
	</form>
</div>
