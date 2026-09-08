import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PropertyCard from './PropertyCard.vue'

vi.mock('@/services/api', () => ({
  authApi: {},
  customerApi: { addFavorite: vi.fn(), removeFavorite: vi.fn(), favorites: vi.fn() },
  publicApi: { meta: vi.fn(), properties: vi.fn() },
}))
vi.mock('@/services/http', () => ({ configureAuth: vi.fn() }))

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push, currentRoute: { value: { fullPath: '/properties' } } }),
  RouterLink: { template: '<a><slot /></a>' },
}))

const property = {
  id: 12,
  name: 'Green Valley Residency',
  slug: 'green-valley-residency-baner-pune',
  builder_name: 'Green Valley Developers',
  city: 'Pune',
  locality: 'Baner',
  cover_image: 'https://example.com/cover.jpg',
  min_price: 7500000,
  max_price: 10500000,
  min_area: 850,
  max_area: 1200,
  bhk_options: [2, 3],
  construction_status: 'UNDER_CONSTRUCTION',
  is_rera_verified: true,
  is_featured: true,
  rera_number: 'P52100047821',
  possession_date: '2027-06-01',
  view_count: 120,
}

function render(overrides = {}) {
  return mount(PropertyCard, {
    props: { property: { ...property, ...overrides } },
    global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } },
  })
}

describe('PropertyCard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('shows the details a buyer scans a listing for', () => {
    const text = render().text()

    expect(text).toContain('Green Valley Residency')
    expect(text).toContain('Baner')
    expect(text).toContain('Pune')
    expect(text).toContain('Green Valley Developers')
    // The price band, in the units buyers use.
    expect(text).toContain('₹75 L - ₹1.05 Cr')
    expect(text).toContain('2, 3 BHK')
  })

  it('marks a RERA-verified project so the badge is visible in a list', () => {
    expect(render().text()).toContain('RERA verified')
    expect(render({ is_rera_verified: false }).text()).not.toContain('RERA verified')
  })

  it('shows the featured badge only when the listing is featured', () => {
    expect(render().text()).toContain('Featured')
    expect(render({ is_featured: false }).text()).not.toContain('Featured')
  })

  it('renders the construction status in plain words', () => {
    expect(render().text()).toContain('Under Construction')
  })

  it('falls back to a placeholder when a listing has no cover image', () => {
    const wrapper = render({ cover_image: '' })
    const src = wrapper.find('img').attributes('src')
    expect(src).toContain('data:image/svg+xml')
  })

  it('sends an anonymous visitor to login when they try to save a property', async () => {
    const wrapper = render()
    await wrapper.find('.save-btn').trigger('click')
    await flushPromises()

    expect(push).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'login' }),
    )
  })

  it('adds and removes the property from the comparison tray', async () => {
    const wrapper = render()
    const compareButton = wrapper.findAll('button').find((b) => b.text() === 'Compare')

    await compareButton.trigger('click')
    expect(wrapper.text()).toContain('Added')

    await wrapper.findAll('button').find((b) => b.text() === 'Added').trigger('click')
    expect(wrapper.text()).toContain('Compare')
  })
})
