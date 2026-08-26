export function signToken(payload: string): string {
	return `signed:${payload}`;
}
