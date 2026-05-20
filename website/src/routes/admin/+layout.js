import { redirect } from '@sveltejs/kit';

export const load = async ({ fetch }) => {
	const token = localStorage.getItem('auth_token');
	if (!token) {
		throw redirect(302, '/login');
	}

	try {
		const res = await fetch('/api/admin/config', {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (!res.ok) {
			localStorage.removeItem('auth_token');
			throw redirect(302, '/login');
		}
		return {};
	} catch (e) {
		if (e?.status === 302) throw e;
		localStorage.removeItem('auth_token');
		throw redirect(302, '/login');
	}
};
