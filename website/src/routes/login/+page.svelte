<script>
	import { goto } from '$app/navigation';

	let username = '';
	let password = '';
	let error = '';
	let loading = false;

	async function handleLogin() {
		if (!username || !password) {
			error = '请输入用户名和密码';
			return;
		}

		loading = true;
		error = '';

		try {
			const res = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password })
			});

			if (!res.ok) {
				error = '用户名或密码错误';
				return;
			}

			const { token } = await res.json();
			localStorage.setItem('auth_token', token);
			goto('/admin');
		} catch (e) {
			error = '连接服务器失败';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex justify-center items-center h-full">
	<div class="card bg-base-200 w-96 shadow-xl">
		<div class="card-body">
			<h2 class="card-title text-2xl justify-center mb-4">管理后台登录</h2>
			{#if error}
				<div class="alert alert-error mb-4">
					<span>{error}</span>
				</div>
			{/if}
			<form on:submit|preventDefault={handleLogin}>
				<div class="form-control mb-3">
					<label class="label" for="username">
						<span class="label-text">用户名</span>
					</label>
					<input
						id="username"
						bind:value={username}
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
						bind:value={password}
						type="password"
						placeholder="输入密码"
						class="input input-bordered w-full"
						required
					/>
				</div>
				<div class="form-control">
					<button type="submit" class="btn btn-primary w-full" disabled={loading}>
						{loading ? '登录中...' : '登录'}
					</button>
				</div>
			</form>
		</div>
	</div>
</div>
