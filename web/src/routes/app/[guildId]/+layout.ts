import { getJSON } from '#lib/api';
import type { Bundle } from '#lib/types';

export async function load({ fetch, params }) {
	const bundle = await getJSON<Bundle>(fetch, `/api/guilds/${params.guildId}`);
	return { bundle };
}
