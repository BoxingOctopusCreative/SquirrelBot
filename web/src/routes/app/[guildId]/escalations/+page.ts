import { getJSON } from '#lib/api';
import type { Escalation } from '#lib/types';

export async function load({ fetch, params }) {
	const body = await getJSON<{ escalations: Escalation[] }>(fetch, `/api/guilds/${params.guildId}/escalations`);
	return { escalations: body.escalations };
}
