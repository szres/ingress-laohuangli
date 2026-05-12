export const load = async ({ cookies }) => {
	const token = cookies.get('auth_token');
	try {
		const res = await fetch(`http://${import.meta.env.VITE_DATA_URL}/api/admin/logs?n=500`, {
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
