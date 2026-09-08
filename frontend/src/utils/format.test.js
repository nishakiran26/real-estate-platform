import { describe, expect, it } from 'vitest'
import {
  bhkLabel,
  formatArea,
  formatPossession,
  formatPrice,
  formatPriceRange,
  humanise,
  pricePerSqft,
} from './format'

describe('price formatting', () => {
  it('uses lakh and crore, the units Indian buyers read', () => {
    expect(formatPrice(7500000)).toBe('₹75 L')
    expect(formatPrice(10500000)).toBe('₹1.05 Cr')
    expect(formatPrice(25000000)).toBe('₹2.5 Cr')
    expect(formatPrice(450000)).toBe('₹4.5 L')
  })

  it('falls back to a plain amount below one lakh', () => {
    expect(formatPrice(50000)).toBe('₹50,000')
  })

  it('says "price on request" rather than showing a zero price', () => {
    expect(formatPrice(0)).toBe('Price on request')
    expect(formatPrice(null)).toBe('Price on request')
    expect(formatPrice(undefined)).toBe('Price on request')
  })

  it('collapses a range when both ends are the same', () => {
    expect(formatPriceRange(7500000, 7500000)).toBe('₹75 L')
    expect(formatPriceRange(7500000, 0)).toBe('₹75 L')
  })

  it('renders a genuine range with both ends', () => {
    expect(formatPriceRange(7500000, 10500000)).toBe('₹75 L - ₹1.05 Cr')
  })
})

describe('area and rate formatting', () => {
  it('formats an area with Indian digit grouping', () => {
    expect(formatArea(1200)).toBe('1,200 sq.ft.')
    expect(formatArea(0)).toBe('-')
  })

  it('computes a per-square-foot rate', () => {
    expect(pricePerSqft(7500000, 850)).toBe('₹8,824/sq.ft.')
    expect(pricePerSqft(0, 850)).toBe('-')
    expect(pricePerSqft(7500000, 0)).toBe('-')
  })
})

describe('possession formatting', () => {
  it('treats a missing or past date as ready to move', () => {
    expect(formatPossession(null)).toBe('Ready to move')
    expect(formatPossession('2020-01-01')).toBe('Ready to move')
  })

  it('shows a month and year for a future handover', () => {
    const future = new Date()
    future.setFullYear(future.getFullYear() + 2)
    const label = formatPossession(future.toISOString())
    expect(label).not.toBe('Ready to move')
    expect(label).toContain(String(future.getFullYear()))
  })
})

describe('label helpers', () => {
  it('turns API enumerations into readable text', () => {
    expect(humanise('UNDER_CONSTRUCTION')).toBe('Under Construction')
    expect(humanise('SITE_VISIT_SCHEDULED')).toBe('Site Visit Scheduled')
    expect(humanise('')).toBe('')
  })

  it('summarises the available configurations', () => {
    expect(bhkLabel([2])).toBe('2 BHK')
    expect(bhkLabel([1, 2, 3])).toBe('1, 3 BHK')
    expect(bhkLabel([])).toBe('')
  })
})
