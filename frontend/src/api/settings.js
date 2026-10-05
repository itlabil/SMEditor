import { request } from './client'

export function getSettings() {
  return request('GET', '/settings')
}

export function updateSettings(values) {
  return request('PUT', '/settings', values)
}

export function checkTools() {
  return request('GET', '/settings/check')
}
