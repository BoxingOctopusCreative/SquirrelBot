import { getJSON } from '#lib/api';
import type { GuildSummary } from '#lib/types';

export async function load({ fetch }) {
	const body = await getJSON<{ guilds: GuildSummary[] }>(fetch, '/api/guilds');
	return { guilds: body.guilds };
}
