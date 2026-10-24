import { getJSON } from '#lib/api';
import type { ModCase } from '#lib/types';

export async function load({ fetch, params }) {
	const body = await getJSON<{ cases: ModCase[] }>(fetch, `/api/guilds/${params.guildId}/cases`);
	return { cases: body.cases };
}
