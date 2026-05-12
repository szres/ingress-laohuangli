import { fail } from '@sveltejs/kit';

export const load = async ({ cookies }) => {
	const token = cookies.get('auth_token');
	const res = await fetch(`http://${import.meta.env.VITE_DATA_URL}/api/admin/config`, {
		headers: { Authorization: `Bearer ${token}` }
	});
	if (!res.ok) {
		return { config: {} };
	}
	const config = await res.json();
	return { config };
};

export const actions = {
	save: async ({ request, cookies }) => {
		const token = cookies.get('auth_token');
		const data = await request.formData();

		const update = {};

		// 收集非空字段
		const fields = ['bot_token', 'admin_id', 'kuma_push_url', 'openai_api_key', 'openai_base_url', 'openai_model', 'admin_username', 'admin_password'];
		for (const field of fields) {
			const value = data.get(field);
			if (value && value.trim() !== '') {
				update[field] = value.trim();
			}
		}

		if (Object.keys(update).length === 0) {
			return fail(400, { error: '没有需要更新的配置' });
		}

		try {
			const res = await fetch(`http://${import.meta.env.VITE_DATA_URL}/api/admin/config`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${token}`
				},
				body: JSON.stringify(update)
			});

			if (!res.ok) {
				return fail(500, { error: '保存配置失败' });
			}

			return { success: '配置已保存' };
		} catch (e) {
			return fail(500, { error: '连接服务器失败' });
		}
	}
};
