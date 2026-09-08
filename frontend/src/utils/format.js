/**
 * Indian formatting helpers. Prices are stored in rupees and shown in the lakh
 * and crore units buyers actually use.
 */

export function formatPrice(value) {
  const amount = Number(value)
  if (!amount || amount <= 0) return 'Price on request'
  if (amount >= 10000000) {
    return `₹${trim(amount / 10000000)} Cr`
  }
  if (amount >= 100000) {
    return `₹${trim(amount / 100000)} L`
  }
  return `₹${new Intl.NumberFormat('en-IN').format(Math.round(amount))}`
}

/** A price band, collapsed to a single value when both ends match. */
export function formatPriceRange(min, max) {
  const low = Number(min) || 0
  const high = Number(max) || 0
  if (!low && !high) return 'Price on request'
  if (!high || low === high) return formatPrice(low)
  return `${formatPrice(low)} - ${formatPrice(high)}`
}

export function formatArea(value, unit = 'sq.ft.') {
  const area = Number(value)
  if (!area) return '-'
  return `${new Intl.NumberFormat('en-IN').format(Math.round(area))} ${unit}`
}

export function formatAreaRange(min, max, unit = 'sq.ft.') {
  const low = Number(min) || 0
  const high = Number(max) || 0
  if (!low && !high) return '-'
  if (!high || low === high) return formatArea(low, unit)
  return `${new Intl.NumberFormat('en-IN').format(Math.round(low))} - ${formatArea(high, unit)}`
}

export function formatNumber(value) {
  return new Intl.NumberFormat('en-IN').format(Number(value) || 0)
}

export function formatDate(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })
}

export function formatDateTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Possession month, which is how builders quote handover. */
export function formatPossession(value) {
  if (!value) return 'Ready to move'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  if (date <= new Date()) return 'Ready to move'
  return date.toLocaleDateString('en-IN', { month: 'short', year: 'numeric' })
}

export function relativeTime(value) {
  if (!value) return ''
  const date = new Date(value)
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000)
  if (seconds < 60) return 'just now'
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  return formatDate(value)
}

/** Turns SNAKE_CASE enumerations into readable labels. */
export function humanise(value) {
  if (!value) return ''
  return String(value)
    .toLowerCase()
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

export function bhkLabel(options = []) {
  if (!options.length) return ''
  if (options.length === 1) return `${options[0]} BHK`
  return `${options[0]}, ${options[options.length - 1]} BHK`
}

export function pricePerSqft(price, area) {
  if (!price || !area) return '-'
  return `₹${formatNumber(Math.round(price / area))}/sq.ft.`
}

export function initials(name) {
  return (name || '?')
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0].toUpperCase())
    .join('')
}

function trim(value) {
  return Number(value.toFixed(2)).toString()
}
