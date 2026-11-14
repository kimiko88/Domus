import { describe, expect, it } from 'vitest'
import deepmerge from 'deepmerge'

describe('vendored deepmerge prototype-poisoning protection', () => {
  it('rejects __proto__ when the target has a null prototype', () => {
    const target = Object.create(null) as Record<string, unknown>
    target.debug = false
    const malicious = JSON.parse('{"__proto__":{"isAdmin":true}}') as Record<string, unknown>

    const merged = deepmerge(target, malicious)

    expect(Object.getPrototypeOf(merged)).toBe(Object.prototype)
    expect(Object.hasOwn(merged, '__proto__')).toBe(false)
    expect(merged.isAdmin).toBeUndefined()
    expect(merged.debug).toBe(false)
  })

  it('rejects nested __proto__ when the target property is null', () => {
    const target = { session: null }
    const malicious = JSON.parse('{"session":{"__proto__":{"isAdmin":true}}}')

    const merged = deepmerge(target, malicious)

    expect(Object.getPrototypeOf(merged.session)).toBe(Object.prototype)
    expect(Object.hasOwn(merged.session, '__proto__')).toBe(false)
    expect(merged.session.isAdmin).toBeUndefined()
  })

  it('does not copy an own __proto__ property from the target', () => {
    const target = JSON.parse('{"__proto__":{"isAdmin":true},"debug":false}')
    const merged = deepmerge(target, { verbose: true })

    expect(Object.getPrototypeOf(merged)).toBe(Object.prototype)
    expect(Object.hasOwn(merged, '__proto__')).toBe(false)
    expect(merged.isAdmin).toBeUndefined()
    expect(merged.debug).toBe(false)
    expect(merged.verbose).toBe(true)
  })

  it('keeps cloned array entries on the normal object prototype', () => {
    const target = { entries: [] }
    const malicious = JSON.parse('{"entries":[{"__proto__":{"isAdmin":true}}]}')

    const merged = deepmerge(target, malicious)

    expect(Object.getPrototypeOf(merged.entries[0])).toBe(Object.prototype)
    expect(Object.hasOwn(merged.entries[0], '__proto__')).toBe(false)
    expect(merged.entries[0].isAdmin).toBeUndefined()
  })
})
