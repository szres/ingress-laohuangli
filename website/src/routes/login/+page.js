export const load = async () => {
	const token = localStorage.getItem('auth_token');
	if (token) {
		return { redirect: '/admin' };
	}
	return {};
};
