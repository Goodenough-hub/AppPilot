import { describe, it, expect } from 'vitest'
import { canAccessHub } from './AuthContext'

describe('canAccessHub', () => {
  it('scope 含 hub 的普通用户放行', () => {
    expect(canAccessHub(['finflow', 'hub'], 'user')).toBe(true)
  })

  it('无 hub scope 的普通用户拒绝', () => {
    expect(canAccessHub(['finflow'], 'user')).toBe(false)
  })

  it('admin 无 hub scope 仍放行（后端伪 scope 直通语义）', () => {
    expect(canAccessHub(['finflow'], 'admin')).toBe(true)
  })

  it('空 scope / null scope 拒绝', () => {
    expect(canAccessHub([], 'user')).toBe(false)
    expect(canAccessHub(null, 'user')).toBe(false)
  })
})
