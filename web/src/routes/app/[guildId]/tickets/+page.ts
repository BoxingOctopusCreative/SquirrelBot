import { getJSON } from '#lib/api';
import type { Ticket } from '#lib/types';

export async function load({ fetch, params }) {
	const body = await getJSON<{ open: Ticket[]; closed: Ticket[] }>(fetch, `/api/guilds/${params.guildId}/tickets`);
	return { open: body.open ?? [], closed: body.closed ?? [] };
}
