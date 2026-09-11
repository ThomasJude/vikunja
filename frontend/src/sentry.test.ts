import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {createMemoryHistory, createRouter} from 'vue-router'
import type {App} from 'vue'
import setupSentry from './sentry'

const sdk = vi.hoisted(() => ({
	init: vi.fn(),
	addEventProcessor: vi.fn(),
	replayIntegration: vi.fn(),
	replay: {name: 'Replay', stop: vi.fn<() => Promise<void>>()},
}))

vi.mock('@sentry/vue', () => ({
	init: sdk.init,
	addEventProcessor: sdk.addEventProcessor,
	browserTracingIntegration: () => ({name: 'BrowserTracing'}),
	replayIntegration: sdk.replayIntegration,
	makeBrowserOfflineTransport: vi.fn(),
	makeFetchTransport: vi.fn(),
	captureMessage: vi.fn(),
}))

async function initialize(path = '/') {
	const router = createRouter({
		history: createMemoryHistory(),
		routes: [
			{path: '/', component: {}},
			{path: '/invite/:token', name: 'user.invite', component: {}},
		],
	})
	await router.push(path)
	await setupSentry({} as App, router)
	return {router, options: sdk.init.mock.calls[0][0]}
}

beforeEach(() => {
	vi.clearAllMocks()
	sdk.replay.stop.mockResolvedValue(undefined)
	sdk.replayIntegration.mockReturnValue(sdk.replay)
})

afterEach(() => {
	window.history.replaceState({}, '', '/')
})

const invitePaths = ['/invite/synthetic-secret', '/Invite/synthetic-secret', '/INVITE/synthetic-secret']

describe('invite telemetry privacy', () => {
	it.each(['beforeSend', 'beforeSendTransaction', 'beforeSendSpan', 'beforeBreadcrumb'].flatMap(hook =>
		invitePaths.map(path => ({hook, path})),
	))(
		'removes invite secrets through $hook for $path while preserving other diagnostics',
		async ({hook, path}) => {
			const {options} = await initialize()
			const payload = {
				message: 'GET https://example.com/api/v2/invite-links/synthetic-secret/register failed',
				request: {url: `https://example.com/vikunja${path}`, headers: {Referer: path}},
				contexts: {vue: {propsData: {inviteToken: 'synthetic-secret'}}},
				data: {'params.token': 'synthetic-secret', 'url.path.parameter.token': 'synthetic-secret', status_code: 500},
				breadcrumbs: [{data: {from: path, to: '/login'}}],
				spans: [{description: 'GET /api/v2/invite-links/synthetic%2Dsecret', data: {'http.response.status_code': 500}}],
				extra: {url: '/api/v2/admin/invite-links/42?page=2', count: 3},
			}
			expect(options[hook]).toBeTypeOf('function')
			const filtered = options[hook](payload, {originalException: new Error('unexpected failure')})
			expect(JSON.stringify(filtered)).not.toContain('synthetic')
			expect(filtered.request.url).toBe(`https://example.com/vikunja${path.replace('synthetic-secret', '[redacted]')}`)
			expect(filtered.extra).toEqual(payload.extra)
			expect(filtered.data.status_code).toBe(500)
			expect(filtered.breadcrumbs[0].data.to).toBe('/login')
			expect(payload.request.url).toContain('synthetic-secret')
		},
	)

	it('scrubs replay navigation records and URL summaries', async () => {
		await initialize()
		expect(sdk.addEventProcessor).toHaveBeenCalled()
		const processor = sdk.addEventProcessor.mock.calls[0][0]
		const summary = processor({type: 'replay_event', urls: ['https://example.com/invite/synthetic-secret']})
		expect(summary.urls).toEqual(['https://example.com/invite/[redacted]'])
		const {beforeAddRecordingEvent} = sdk.replayIntegration.mock.calls[0][0]
		expect(beforeAddRecordingEvent).toBeTypeOf('function')
		const recording = beforeAddRecordingEvent({type: 5, data: {payload: {name: '/invite/synthetic-secret'}}})
		expect(recording.data.payload.name).toBe('/invite/[redacted]')
	})

	it('keeps replay disabled if navigation leaves an invite during SDK import', async () => {
		const router = createRouter({history: createMemoryHistory(), routes: [{path: '/', component: {}}]})
		await router.push('/')
		window.history.replaceState({}, '', '/invite/synthetic-secret')
		const setup = setupSentry({} as App, router)
		window.history.replaceState({}, '', '/')
		await setup
		expect(sdk.init.mock.calls[0][0].integrations).not.toContain(sdk.replay)
	})

	it.each(invitePaths)('excludes replay when initialized on %s', async path => {
		const {options} = await initialize(path)
		expect(options.integrations).not.toContain(sdk.replay)
	})

	it.each(invitePaths)('excludes replay for %s under a frontend subpath', async path => {
		window.history.replaceState({}, '', `/vikunja${path}`)
		const {options} = await initialize()
		expect(options.integrations).not.toContain(sdk.replay)
	})

	it.each(invitePaths)('stops replay before navigating to %s and never resumes it', async path => {
		let finishStop!: () => void
		const stopped = new Promise<void>(resolve => { finishStop = resolve })
		sdk.replay.stop.mockReturnValue(stopped)
		const {router, options} = await initialize()
		expect(options.integrations).toContain(sdk.replay)
		const navigation = router.push(path)
		await vi.waitFor(() => expect(sdk.replay.stop).toHaveBeenCalled())
		expect(router.currentRoute.value.path).toBe('/')
		finishStop()
		await navigation
		expect(router.currentRoute.value.path).toBe(path)
		await router.push('/')
		expect(sdk.replay.stop).toHaveBeenCalledTimes(1)
		expect(sdk.init).toHaveBeenCalledTimes(1)
	})
})
