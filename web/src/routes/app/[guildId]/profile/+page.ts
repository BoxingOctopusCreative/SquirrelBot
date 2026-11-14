import { getJSON } from '#lib/api';
import type { BotProfile } from '#lib/types';

export async function load({ fetch, params }) {
	const profile = await getJSON<BotProfile>(fetch, `/api/guilds/${params.guildId}/profile`);
	return { profile };
}
