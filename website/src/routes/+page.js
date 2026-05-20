export const load = async ({ fetch }) => {
	const res = await fetch('/api/cache');
	if (!res.ok) {
		return { date: '', today: {}, caches: {} };
	}
	return res.json();
};
