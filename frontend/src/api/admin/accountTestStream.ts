export interface AccountTestStreamRequest {
  modelId: string
  prompt?: string
}

export interface AccountTestStreamEvent {
  type: string
  text?: string
  model?: string
  success?: boolean
  error?: string
  image_url?: string
  mime_type?: string
}

export interface AccountTestStreamOptions {
  signal?: AbortSignal
  onEvent?: (event: AccountTestStreamEvent) => void
  fetchImpl?: typeof fetch
  authToken?: string | null
}

export interface AccountTestStreamResult {
  success: boolean
  error?: string
}

const buildAuthToken = (authToken?: string | null) => {
  if (typeof authToken === 'string') return authToken
  if (typeof localStorage === 'undefined') return null
  return localStorage.getItem('auth_token')
}

export async function runAccountTestStream(
  accountId: number,
  request: AccountTestStreamRequest,
  options: AccountTestStreamOptions = {}
): Promise<AccountTestStreamResult> {
  const fetchImpl = options.fetchImpl ?? fetch
  const authToken = buildAuthToken(options.authToken)

  const response = await fetchImpl(`/api/v1/admin/accounts/${accountId}/test`, {
    method: 'POST',
    headers: {
      ...(authToken ? { Authorization: `Bearer ${authToken}` } : {}),
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      model_id: request.modelId,
      prompt: request.prompt ?? ''
    }),
    signal: options.signal
  })

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }

  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error('No response body')
  }

  const decoder = new TextDecoder()
  let buffer = ''
  let completed = false
  let finalResult: AccountTestStreamResult = {
    success: false,
    error: 'Test stream ended unexpectedly'
  }

  const applyEvent = (event: AccountTestStreamEvent) => {
    options.onEvent?.(event)

    if (event.type === 'test_complete') {
      completed = true
      finalResult = event.success
        ? { success: true }
        : { success: false, error: event.error || 'Test failed' }
    }

    if (event.type === 'error') {
      completed = true
      finalResult = {
        success: false,
        error: event.error || 'Unknown error'
      }
    }
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const line of lines) {
      if (!line.startsWith('data: ')) continue
      const jsonStr = line.slice(6).trim()
      if (!jsonStr) continue
      try {
        const event = JSON.parse(jsonStr) as AccountTestStreamEvent
        applyEvent(event)
      } catch (error) {
        console.error('Failed to parse SSE event:', error)
      }
    }
  }

  if (!completed) {
    throw new Error(finalResult.error || 'Test stream ended unexpectedly')
  }

  return finalResult
}

export default runAccountTestStream
