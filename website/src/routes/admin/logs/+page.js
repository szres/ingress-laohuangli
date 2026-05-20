import { redirect } from '@sveltejs/kit';

export const load = async ({ fetch }) => {
	const token = localStorage.getItem('auth_token');
	if (!token) {
		throw redirect(302, '/login');
	}

	try {
		const res = await fetch('/api/admin/logs?n=500', {
			headers: { Authorization: `Bearer ${token}` }
		});
		if (!res.ok) {
			return { logs: [] };
		}
		const data = await res.json();
		return { logs: data.lines || [] };
	} catch (e) {
		return { logs: [] };
	}
};
