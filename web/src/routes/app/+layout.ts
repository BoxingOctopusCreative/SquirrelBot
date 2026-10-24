import { getJSON } from '#lib/api';
import type { Me } from '#lib/types';

export async function load({ fetch }) {
	const me = await getJSON<Me>(fetch, '/api/me');
	return { me };
}
