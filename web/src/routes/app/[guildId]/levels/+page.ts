import { getJSON } from '#lib/api';
import type { LevelEvent, LevelStanding } from '#lib/types';

export async function load({ fetch, params }) {
	const body = await getJSON<{ leaderboard: LevelStanding[]; events: LevelEvent[]; total: number }>(
		fetch,
		`/api/guilds/${params.guildId}/levels`
	);
	return {
		leaderboard: body.leaderboard ?? [],
		events: body.events ?? [],
		total: body.total ?? 0
	};
}
