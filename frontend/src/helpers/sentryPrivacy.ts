const INVITE_URL = /(\/(?:api\/v2\/invite-links|invite)\/)[^/?#\s"'<>]+/gi

export function isInvitePage(path: string): boolean {
	return /(?:^|\/)invite\/[^/?#]+/i.test(path)
}

export function scrubInviteSecrets<T>(payload: T): T {
	const seen = new WeakMap<object, unknown>()

	function scrub(value: unknown): unknown {
		if (typeof value === 'string') {
			return value.replace(INVITE_URL, '$1[redacted]')
		}
		if (!value || typeof value !== 'object') {
			return value
		}
		if (seen.has(value)) {
			return seen.get(value)
		}
		if (Array.isArray(value)) {
			const result: unknown[] = []
			seen.set(value, result)
			for (const item of value) result.push(scrub(item))
			return result
		}
		if (Object.getPrototypeOf(value) !== Object.prototype && Object.getPrototypeOf(value) !== null) {
			return value
		}
		const result: Record<string, unknown> = {...value}
		seen.set(value, result)
		for (const [key, entry] of Object.entries(value)) {
			result[key] = key === 'inviteToken' || key === 'params.token' || key === 'url.path.parameter.token'
				? '[redacted]'
				: scrub(entry)
		}
		return result
	}

	return scrub(payload) as T
}
