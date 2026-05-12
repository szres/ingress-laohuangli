export const load = async ({ fetch }) => {
	const res = await fetch(`http://${import.meta.env.VITE_DATA_URL}/api/cache`);
	if (!res.ok) {
		return { date: '', today: {}, caches: {} };
	}
	return res.json();
};
