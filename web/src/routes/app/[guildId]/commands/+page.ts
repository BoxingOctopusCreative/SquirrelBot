import { getJSON } from '#lib/api';
import type { CustomCommand } from '#lib/types';

export async function load({ fetch, params }) {
	const body = await getJSON<{ commands: CustomCommand[] }>(fetch, `/api/guilds/${params.guildId}/commands`);
	return { commands: body.commands };
}
