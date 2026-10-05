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
