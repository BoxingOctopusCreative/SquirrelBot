import { getJSON } from '#lib/api';
import type { ReactionRole } from '#lib/types';

export async function load({ fetch, params }) {
	const body = await getJSON<{ reactionRoles: ReactionRole[] }>(
		fetch,
		`/api/guilds/${params.guildId}/reaction-roles`
	);
	return { reactionRoles: body.reactionRoles };
}
