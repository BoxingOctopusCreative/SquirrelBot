import { getJSON } from '#lib/api';
import type { Announcement, AnnouncementTemplate } from '#lib/types';

export async function load({ fetch, params }) {
	const [announcements, templates] = await Promise.all([
		getJSON<{ announcements: Announcement[] }>(fetch, `/api/guilds/${params.guildId}/announcements`),
		getJSON<{ templates: AnnouncementTemplate[] }>(fetch, `/api/guilds/${params.guildId}/announcement-templates`)
	]);
	return { announcements: announcements.announcements, templates: templates.templates };
}
