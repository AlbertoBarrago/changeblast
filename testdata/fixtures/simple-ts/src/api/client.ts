import { authMiddleware } from '../auth/middleware';

export function handleRequest(req: unknown): void {
	authMiddleware(req);
}
