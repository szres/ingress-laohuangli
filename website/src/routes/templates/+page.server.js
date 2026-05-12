export const load = async () => {
	const baseUrl = `http://${import.meta.env.VITE_DATA_URL}`;
	const [templatesRes, entriesRes] = await Promise.all([
		fetch(`${baseUrl}/api/templates`),
		fetch(`${baseUrl}/api/entries`)
	]);

	const templates = templatesRes.ok ? await templatesRes.json() : {};
	const entriesData = entriesRes.ok ? await entriesRes.json() : { entries: [], entries_user: [] };

	return {
		templates,
		entrys: entriesData.entries || [],
		entrysUser: entriesData.entries_user || []
	};
};
