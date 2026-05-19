export const load = async ({ fetch }) => {
	const [templatesRes, entriesRes] = await Promise.all([
		fetch('/api/templates'),
		fetch('/api/entries')
	]);

	const templates = templatesRes.ok ? await templatesRes.json() : {};
	const entriesData = entriesRes.ok ? await entriesRes.json() : { entries: [], entries_user: [] };

	return {
		templates,
		entrys: entriesData.entries || [],
		entrysUser: entriesData.entries_user || []
	};
};
