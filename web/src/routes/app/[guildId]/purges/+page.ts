import { getJSON } from '#lib/api';
import type { ChannelPurge } from '#lib/types';

export async function load({ fetch, params }) {
	const body = await getJSON<{ enabled: boolean; schedules: ChannelPurge[] }>(
		fetch,
		`/api/guilds/${params.guildId}/purges`
	);
	return { schedules: body.schedules };
}
