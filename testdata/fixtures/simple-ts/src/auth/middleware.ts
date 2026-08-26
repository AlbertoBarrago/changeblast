import { signToken } from './token';

export function authMiddleware(req: unknown): void {
	signToken('req');
}
