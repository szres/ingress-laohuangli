import { redirect, fail } from '@sveltejs/kit';

export const load = async ({ cookies }) => {
	const token = cookies.get('auth_token');
	if (token) {
		throw redirect(302, '/admin');
	}
	return {};
};

export const actions = {
	default: async ({ request, cookies }) => {
		const data = await request.formData();
		const username = data.get('username');
		const password = data.get('password');

		if (!username || !password) {
			return fail(400, { error: '请输入用户名和密码' });
		}

		try {
			const res = await fetch(`http://${import.meta.env.VITE_DATA_URL}/api/auth/login`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ username, password })
			});

			if (!res.ok) {
				return fail(401, { error: '用户名或密码错误' });
			}

			const { token } = await res.json();
			cookies.set('auth_token', token, {
				path: '/',
				httpOnly: true,
				sameSite: 'strict',
				maxAge: 60 * 60 * 24 // 24 hours
			});
		} catch (e) {
			return fail(500, { error: '连接服务器失败' });
		}

		throw redirect(302, '/admin');
	}
};
