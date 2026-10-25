import { error, redirect } from '@sveltejs/kit';

type LoadFetch = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

export async function getJSON<T>(fetcher: LoadFetch, path: string): Promise<T> {
	const response = await fetcher(path);
	if (response.status === 401) redirect(307, '/');
	if (!response.ok) {
		const body = (await response.json().catch(() => ({}))) as { error?: string };
		error(response.status, body.error || 'Could not load this page.');
	}
	return response.json() as Promise<T>;
}

export async function sendJSON<T>(path: string, method: string, body?: unknown): Promise<T> {
	const response = await fetch(path, {
		method,
		credentials: 'same-origin',
		headers: body === undefined ? undefined : { 'content-type': 'application/json' },
		body: body === undefined ? undefined : JSON.stringify(body)
	});
	if (response.status === 204) return undefined as T;
	const payload = (await response.json().catch(() => ({}))) as { error?: string };
	if (!response.ok) throw new Error(payload.error || 'Could not save.');
	return payload as T;
}

export async function sendForm<T>(path: string, method: string, body: FormData): Promise<T> {
	const response = await fetch(path, {
		method,
		credentials: 'same-origin',
		body
	});
	const payload = (await response.json().catch(() => ({}))) as { error?: string };
	if (!response.ok) throw new Error(payload.error || 'Could not save.');
	return payload as T;
}

export function when(value?: string | null): string {
	if (!value) return '';
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return '';
	return date.toLocaleString();
}
