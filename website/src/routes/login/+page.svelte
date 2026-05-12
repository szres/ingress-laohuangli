<script>
	export let form;
</script>

<div class="flex justify-center items-center h-full">
	<div class="card bg-base-200 w-96 shadow-xl">
		<div class="card-body">
			<h2 class="card-title text-2xl justify-center mb-4">管理后台登录</h2>
			{#if form?.error}
				<div class="alert alert-error mb-4">
					<span>{form.error}</span>
				</div>
			{/if}
			<form method="POST">
				<div class="form-control mb-3">
					<label class="label" for="username">
						<span class="label-text">用户名</span>
					</label>
					<input
						id="username"
						name="username"
						type="text"
						placeholder="输入用户名"
						class="input input-bordered w-full"
						required
					/>
				</div>
				<div class="form-control mb-6">
					<label class="label" for="password">
						<span class="label-text">密码</span>
					</label>
					<input
						id="password"
						name="password"
						type="password"
						placeholder="输入密码"
						class="input input-bordered w-full"
						required
					/>
				</div>
				<div class="form-control">
					<button type="submit" class="btn btn-primary w-full">登录</button>
				</div>
			</form>
		</div>
	</div>
</div>
