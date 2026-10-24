import { getJSON } from '#lib/api';
import type { GateMember } from '#lib/types';

export async function load({ fetch, params }) {
	const body = await getJSON<{ pending: GateMember[]; recent: GateMember[] }>(
		fetch,
		`/api/guilds/${params.guildId}/gate/members`
	);
	return { pending: body.pending, recent: body.recent };
}
