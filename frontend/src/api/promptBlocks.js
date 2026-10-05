import { request } from './client'

export function listGameModes() {
  return request('GET', '/game-modes')
}

export function getPromptBlock(code) {
  return request('GET', `/prompt-blocks/${code}`)
}

export function updatePromptBlock(code, body, categories) {
  return request('PUT', `/prompt-blocks/${code}`, { body, categories })
}

export function resetPromptBlock(code) {
  return request('POST', `/prompt-blocks/${code}/reset`)
}
