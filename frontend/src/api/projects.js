import { request } from './client'

export function listProjects() {
  return request('GET', '/projects')
}

export function getProject(id) {
  return request('GET', `/projects/${id}`)
}

export function createProject(payload) {
  return request('POST', '/projects', payload)
}

export function deleteProject(id) {
  return request('DELETE', `/projects/${id}`)
}

export function retryDownload(id) {
  return request('POST', `/projects/${id}/download`)
}

export function thumbnailUrl(id) {
  return `/api/projects/${id}/thumbnail`
}

export function videoUrl(id) {
  return `/api/projects/${id}/video`
}

export function retryTranscribe(id, transcriptLang) {
  return request('POST', `/projects/${id}/transcribe`, { transcript_lang: transcriptLang })
}

export function transcriptUrl(id, format) {
  return `/api/projects/${id}/transcript?format=${format}`
}

export function getPrompt(id) {
  return request('GET', `/projects/${id}/prompt`)
}

export function openFolder(id) {
  return request('POST', `/projects/${id}/open-folder`)
}
