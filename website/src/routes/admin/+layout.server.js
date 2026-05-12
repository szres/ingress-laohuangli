import { redirect } from '@sveltejs/kit';

export const load = async ({ cookies }) => {
	const token = cookies.get('auth_token');
	if (!token) {
		throw redirect(302, '/login');
	}

	// 验证 token 是否有效
	try {
		const res = await fetch(`http://${import.meta.env.VITE_DATA_URL}/api/admin/config`, {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (!res.ok) {
			cookies.delete('auth_token', { path: '/' });
			throw redirect(302, '/login');
		}
		return {};
	} catch (e) {
		if (e?.status === 302) throw e;
		cookies.delete('auth_token', { path: '/' });
		throw redirect(302, '/login');
	}
};
